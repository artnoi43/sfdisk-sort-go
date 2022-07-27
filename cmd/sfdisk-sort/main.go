package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// These are just strings for printing/checking
const (
	stdinFlag      = "-stdin" // CLI
	beforeSort     = "# sfdisk -d ORIGINAL DUMP OUTPUT\n"
	sortSuccessful = "# sfdisk -d SORTED OUTPUT:\n"
	bottomLine     = "# See https://github.com/artnoi43/sfdisk-sort-go/blob/main/README.md to see what to do whith this output\n"
)

type partition struct {
	// For partition number sorting
	designation int // 1
	startBlock  int // 2048

	// String values are only used to parse/reconstruct `sfdisk -d` output lines
	name      string // "/dev/sda1"
	size      string // "69000," - note: has trailing comma
	partType  string // "type=0F69," - note: has trailing comma
	uuid      string // "uuid=0F69," - note: MAY HAVE trailing comma if there is extraInfo
	extraInfo string // e.g. name or label
}

type partitions []partition

type disk struct {
	name       string
	header     string
	partitions partitions
}

func (part *partition) String() string {
	partString := fmt.Sprintf("%s : start= %d, size= %s %s %s", part.name, part.startBlock, part.size, part.partType, part.uuid)
	if part.extraInfo != "" {
		return fmt.Sprintf("%s %s", partString, part.extraInfo)
	}
	return partString
}

func (parts *partitions) String() string {
	var s string
	for i, part := range *parts {
		if i == 0 {
			s = part.String()
			continue
		}
		s = fmt.Sprintf("%s\n%s", s, part.String())
	}
	return s
}

func (d *disk) String() string {
	return fmt.Sprintf("%s\n%s", d.header, d.partitions.String())
}

func (p partitions) Less(i, j int) bool {
	return p[i].startBlock < p[j].startBlock
}

func (p partitions) Len() int {
	return len(p)
}

func (p partitions) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func main() {
	// os.Args was initialized from package os when this program runs
	args := os.Args
	if len(args) <= 1 {
		os.Stderr.Write([]byte("invalid number of arguments\n"))
		os.Exit(1)
	}

	if args[1] != stdinFlag {
		// This program will call sfdisk an
		callSfdisk(args)
		return
	}
	// Read sfdisk output from stdin
	readStdin()
}

// callSfdisk uses os/exec to execute `sfdisk -d`, then collects and parses sfdisk stdout output to struct disk,
// and then passes it to partseAndRearrange.
func callSfdisk(args []string) {
	disks := args[1:]
	for _, disk := range disks {
		// Get current partition table
		cmdArgs := fmt.Sprintf("-d %s", disk)
		cmdSfdisk := exec.Command("sfdisk", strings.Fields(cmdArgs)...)
		var sfdiskDumpOutput = new(bytes.Buffer)
		cmdSfdisk.Stdout = sfdiskDumpOutput
		if err := cmdSfdisk.Run(); err != nil {
			os.Stderr.Write([]byte(fmt.Sprintf("failed to get current partition table for %s\n%s\n", disk, err.Error())))
			os.Exit(1)
		}

		cmdSfdiskOut := sfdiskDumpOutput.String()
		cmdSfdiskOut = prependComment(cmdSfdiskOut, "#")
		os.Stdout.Write([]byte(beforeSort + cmdSfdiskOut))

		prettyDisk, err := parseAndRearrange(sfdiskDumpOutput)
		if err != nil {
			os.Stderr.Write([]byte(fmt.Sprintf("error parsing and rearranging sfdisk partitions: %s\n", err.Error())))
			os.Exit(2)
		}
		os.Stdout.Write([]byte(sortSuccessful + prettyDisk.String()))
	}

	os.Stdout.Write([]byte(fmt.Sprintf("\n%s\n", bottomLine)))
}

// readStdin reads `sfdisk -d` output from stdin, and then passes it tp parseAndRearrange
func readStdin() {
	stdin := bufio.NewReader(os.Stdin)
	prettyDisk, err := parseAndRearrange(stdin)
	if err != nil {
		os.Stderr.Write([]byte("failed to parse sfdisk output\n"))
	}

	os.Stdout.Write([]byte(sortSuccessful + prettyDisk.String()))
	os.Stdout.Write([]byte(fmt.Sprintf("\n%s\n", bottomLine)))
}

func parseAndRearrange(sfdiskOutput io.Reader) (*disk, error) {
	// Read original partition table to memory
	parsedDisk, err := parseSfdiskDumpOutput(sfdiskOutput)
	if err != nil {
		os.Stderr.Write([]byte(fmt.Sprintf("error parsing sfdisk output: %s\n", err.Error())))
		return nil, err
	}

	// Re-assign designation
	prettyPartitions, err := redesignatePartitions(parsedDisk.partitions)
	if err != nil {
		os.Stderr.Write([]byte(fmt.Sprintf("error rearranging partitions for disk %s\n", parsedDisk.name)))
		return nil, err
	}

	// Return a new disk
	return &disk{
		header:     parsedDisk.header,
		partitions: prettyPartitions,
	}, nil
}

// parseSfdiskOutputFile parses each line of sfdisk stdout output into a header string and an instance of struct partition
func parseSfdiskDumpOutput(sfdiskOutput io.Reader) (*disk, error) {
	var baseDiskName string
	var parts partitions
	var header string
	scanner := bufio.NewScanner(sfdiskOutput)
	for scanner.Scan() {
		line := scanner.Text()
		lineParsed := strings.Fields(line)
		var partExtraInfo string
		// TODO: Fix this mess
		if l := len(lineParsed); l < 8 {
			if line == "" {
				continue
			}
			if header == "" {
				// Blank line: line is iterated line by line, so "\n" is not included
				header = line
			} else {
				header = header + "\n" + line
				if l == 2 && lineParsed[0] == "device:" {
					baseDiskName = lineParsed[1]
				}
			}
			continue
		} else if l >= 9 {
			// TODO: fix this
			partExtraInfoSlice := lineParsed[8:]
			var ending string
			for i, text := range partExtraInfoSlice {
				if i != 0 {
					ending = fmt.Sprintf("%s %s", ending, text)
				}
			}
			partExtraInfo = partExtraInfoSlice[0] + ending
		}
		if baseDiskName == "" {
			return nil, fmt.Errorf("failed to get base disk name from sfdisk output")
		}
		if lineParsed[1] != ":" {
			continue
		}
		partName := lineParsed[0]
		partStartStr := lineParsed[3]
		partStartBlock, err := strconv.Atoi(strings.Split(partStartStr, ",")[0]) // trim trailing ","
		if err != nil {
			return nil, fmt.Errorf("failed to parse start block to int for partition %s: %s", partName, err.Error())
		}
		partDesignationStr := strings.Split(partName, baseDiskName)[1]
		var partDesignation int
		if strings.Contains(partName, "nvme") && strings.Contains(partDesignationStr, "p") {
			actualDesignation := strings.Split(partDesignationStr, "p")[1] // behind 'p'
			partDesignation, err = strconv.Atoi(actualDesignation)
			if err != nil {
				return nil, fmt.Errorf("failed to parse partition designation for partition %s: %s", partName, err.Error())
			}
		} else {
			partDesignation, err = strconv.Atoi(partDesignationStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse partition designation for partition %s: %s", partName, err.Error())
			}
		}
		if partDesignation == 0 {
			return nil, fmt.Errorf("partition designation number is 0")
		}

		partSize := lineParsed[5]
		partType := lineParsed[6]
		partUuid := lineParsed[7]

		parts = append(parts, partition{
			designation: partDesignation,
			startBlock:  partStartBlock,
			name:        partName,
			size:        partSize,
			partType:    partType,
			uuid:        partUuid,
			extraInfo:   partExtraInfo,
		})
	}

	return &disk{
		name:       baseDiskName,
		header:     header,
		partitions: parts,
	}, nil
}

// redesignatePartitions sorts "parts" and returns an updated slice 'partitions' based on the sorted slice indices.
func redesignatePartitions(parts partitions) (partitions, error) {
	sort.Sort(parts)
	var ret_parts partitions // for return
	for i, part := range parts {
		var isNvme bool
		if strings.Contains(part.name, "nvme") {
			isNvme = true
		}

		oldDesignation := part.designation
		oldDesignationStr := fmt.Sprintf("%d", oldDesignation)
		if isNvme {
			oldDesignationStr = fmt.Sprintf("p%d", oldDesignation)
		}
		basePartName := strings.Split(part.name, oldDesignationStr)[0]
		newDesignation := i + 1
		part.designation = newDesignation
		if isNvme {
			part.name = fmt.Sprintf("%sp%d", basePartName, newDesignation)
		} else {
			part.name = fmt.Sprintf("%s%d", basePartName, newDesignation)
		}
		ret_parts = append(ret_parts, part)
	}

	return ret_parts, nil
}

func prependComment(text string, escapeToken string) string {
	scanner := bufio.NewScanner(strings.NewReader(text))
	var s string
	for scanner.Scan() {
		commentedLine := fmt.Sprintf("%s%s\n", escapeToken, scanner.Text())
		if s == "" {
			s = commentedLine
		} else {
			s = s + commentedLine
		}
	}
	return s
}
