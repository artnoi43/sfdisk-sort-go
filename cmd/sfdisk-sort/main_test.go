package main

import (
	"bufio"
	"reflect"
	"strings"
	"testing"
)

func TestSfdiskSortHelper(t *testing.T) {
	t.Run("test_parse_sfdisk_output", testParseSfdiskDumpOutput)
	t.Run("test_sort_partitions", testSortPartitions)
}

func testParseSfdiskDumpOutput(t *testing.T) {
	disks := map[string]map[string]disk{
		"/dev/sda": {
			`label: gpt
label-id: 12345678-2345-6969-3264-A55555555555
device: /dev/sda
unit: sectors
first-lba: 2048
last-lba: 976773134
sector-size: 512

/dev/sda1 : start=        2048, size=      409600, type=C12A7328-F81F-11D2-BA4B-00A0C93EC93B, uuid=AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE
/dev/sda2 : start=      411648, size=    67108864, type=0FC63DAF-8483-4772-8E79-3D69D8477DE4, uuid=FFFFFFFF-GGGG-HHHH-IIII-JJJJJJJJJJJJ
/dev/sda3 : start=    67520512, size=    33554432, type=0FC63DAF-8483-4772-8E79-3D69D8477DE4, uuid=KKKKKKKK-LLLL-MMMM-NNNN-OOOOOOOOOOOO
/dev/sda4 : start=   101074944, size=   875698191, type=6A85CF4D-1DD2-11B2-99A6-080020736631, uuid=PPPPPPPP-QQQQ-RRRR-SSSS-TTTTTTTTTTTT`: disk{
				header: `label: gpt
label-id: 12345678-2345-6969-3264-A55555555555
device: /dev/sda
unit: sectors
first-lba: 2048
last-lba: 976773134
sector-size: 512`,
				partitions: partitions{
					partition{designation: 1, startBlock: 2048, name: "/dev/sda1", size: "409600,", partType: "type=C12A7328-F81F-11D2-BA4B-00A0C93EC93B,", uuid: "uuid=AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"},
					partition{designation: 2, startBlock: 411648, name: "/dev/sda2", size: "67108864,", partType: "type=0FC63DAF-8483-4772-8E79-3D69D8477DE4,", uuid: "uuid=FFFFFFFF-GGGG-HHHH-IIII-JJJJJJJJJJJJ"},
					partition{designation: 3, startBlock: 67520512, name: "/dev/sda3", size: "33554432,", partType: "type=0FC63DAF-8483-4772-8E79-3D69D8477DE4,", uuid: "uuid=KKKKKKKK-LLLL-MMMM-NNNN-OOOOOOOOOOOO"},
					partition{designation: 4, startBlock: 101074944, name: "/dev/sda4", size: "875698191,", partType: "type=6A85CF4D-1DD2-11B2-99A6-080020736631,", uuid: "uuid=PPPPPPPP-QQQQ-RRRR-SSSS-TTTTTTTTTTTT"},
				},
			},
		},
		"/dev/nvme0n1": {
			`label: gpt
label-id: 12345678-F226-1234-5678-E55555555555
device: /dev/nvme0n1
unit: sectors
first-lba: 2048
last-lba: 60088286
sector-size: 512
			
/dev/nvme0n1p1 : start=  2048,    size= 60086239, type=0FC63DAF-8483-4772-8E79-3D69D8477DE4, uuid=AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE,    it ain't me    babe
/dev/nvme0n1p2 : start=  60088287 size= 60086239, type=0FC63DAF-8483-4772-8E79-3D69D8477DE4, uuid=FFFFFFFF-GGGG-HHHH-IIII-JJJJJJJJJJJJ, it's me babe
			`: disk{
				header: `label: gpt
label-id: 12345678-F226-1234-5678-E55555555555
device: /dev/nvme0n1
unit: sectors
first-lba: 2048
last-lba: 60088286
sector-size: 512`,
				partitions: partitions{
					partition{designation: 1, startBlock: 2048, name: "/dev/nvme0n1p1", size: "60086239,", partType: "type=0FC63DAF-8483-4772-8E79-3D69D8477DE4,", uuid: "uuid=AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE,", extraInfo: "it ain't me babe"},
					partition{designation: 2, startBlock: 60088287, name: "/dev/nvme0n1p2", size: "60086239,", partType: "type=0FC63DAF-8483-4772-8E79-3D69D8477DE4,", uuid: "uuid=FFFFFFFF-GGGG-HHHH-IIII-JJJJJJJJJJJJ,", extraInfo: "it's me babe"},
				},
			},
		},
	}

	for diskName, test := range disks {
		t.Logf("Current diskName: %s", diskName)

		for expectedSfDiskOutput, expectedDisk := range test {
			parsedDisk, err := parseSfdiskDumpOutput(strings.NewReader(expectedSfDiskOutput))
			if err != nil {
				t.Error(err.Error())
			}

			t.Logf("expected disk header:\n%s", expectedDisk.header)
			t.Logf("result disk header:\n%s", parsedDisk.header)
			resultHeaderScanner := bufio.NewScanner(strings.NewReader(parsedDisk.header))
			expectedHeaderScanner := bufio.NewScanner(strings.NewReader(expectedDisk.header))

			// Compare line by line
			for expectedHeaderScanner.Scan() {
				var bothGood bool // false
				if resultHeaderScanner.Scan() {
					bothGood = true
				}
				if bothGood {
					resultHeaderLine := resultHeaderScanner.Text()
					expectedHeaderLine := expectedHeaderScanner.Text()
					resultHeaderFields := strings.Fields(resultHeaderLine)
					expectedHeaderFields := strings.Fields(expectedHeaderLine)
					if l0, l1 := len(resultHeaderFields), len(expectedHeaderFields); l0 != l1 {
						t.Log("different line numbers for header")
						t.Log("result lines", l0)
						t.Log("expected lines", l1)
					}
					// Compare field by field
					for i, resultHeader := range resultHeaderFields {
						if resultHeader != expectedHeaderFields[i] {
							t.Log("unexpected header line")
							t.Log("result header")
							t.Log(resultHeader)
							t.Log("expected result header")
							t.Log(expectedHeaderFields[i])
						}
					}
				}
			}

			if !reflect.DeepEqual(expectedDisk.partitions, parsedDisk.partitions) {
				for i, expectedPart := range expectedDisk.partitions {
					resultPart := parsedDisk.partitions[i]
					t.Log("designation matched", expectedPart.designation == resultPart.designation)
					t.Log("startBlock matched", expectedPart.startBlock == resultPart.startBlock)
					t.Log("name matched", expectedPart.name == resultPart.name)
					t.Log("partType matched", expectedPart.partType == resultPart.partType)
					t.Log("uuid matched", expectedPart.uuid == resultPart.uuid)
					t.Log("extraInfo matched", expectedPart.extraInfo == resultPart.extraInfo)
				}
				t.Fatalf("unexpected result")
			}
		}
	}
}

func testSortPartitions(t *testing.T) {
	sda1 := partition{designation: 1, startBlock: 2048, name: "/dev/sda1", size: "69000,", partType: "type=0x69,", uuid: "uuid=0x1,", extraInfo: "hee=kuy 1"}
	sda2 := partition{designation: 2, startBlock: 2048 + 6900, name: "/dev/sda2", size: "69000,", partType: "type=0x70,", uuid: "uuid=0x2,", extraInfo: "hee=kuy 2"}
	sda3 := partition{designation: 3, startBlock: 2048 + 6900 + 6900, name: "/dev/sda3", size: "69000,", partType: "type=0x70,", uuid: "uuid=0x3,", extraInfo: "hee=kuy 3"}
	sda4 := partition{designation: 4, startBlock: 2048 + 6900 + 6900 + 6900, name: "/dev/sda4", size: "69000,", partType: "type=0x70,", uuid: "uuid=0x4,", extraInfo: "hee=kuy 4"}
	sda := partitions{sda1, sda2, sda3, sda4}

	nvme0n1p1 := partition{designation: 1, startBlock: 2048, name: "/dev/nvme0n1p1", size: "69000,", partType: "type=0x69,", uuid: "uuid=0x1,", extraInfo: "hee=kuy 1"}
	nvme0n1p2 := partition{designation: 2, startBlock: 2048 + 6900, name: "/dev/nvme0n1p2", size: "69000,", partType: "type=0x70,", uuid: "uuid=0x2,", extraInfo: "hee=kuy 2"}
	nvme0n1p3 := partition{designation: 3, startBlock: 2048 + 6900 + 6900, name: "/dev/nvme0n1p3", size: "69000,", partType: "type=0x70,", uuid: "uuid=0x3,", extraInfo: "hee=kuy 3"}
	nvme0n1p4 := partition{designation: 4, startBlock: 2048 + 6900 + 6900 + 6900, name: "/dev/nvme0n1p4", size: "69000,", partType: "type=0x70,", uuid: "uuid=0x4,", extraInfo: "hee=kuy 4"}
	nvme0n1 := partitions{nvme0n1p1, nvme0n1p2, nvme0n1p3, nvme0n1p4}

	uglySda := partitions{
		// Should be sda1
		makeUgly(sda1, 1, "/dev/sda1"),
		// Should be sda2
		makeUgly(sda2, 3, "/dev/sda3"),
		// Should be sda3
		makeUgly(sda3, 2, "/dev/sda2"),
		// should be sda4
		makeUgly(sda4, 4, "/dev/sda4"),
	}
	uglyNvme0n1 := partitions{
		makeUgly(nvme0n1p1, 1, "/dev/nvme0n1p1"),
		makeUgly(nvme0n1p2, 9, "/dev/nvme0n1p9"),
		makeUgly(nvme0n1p3, 6, "/dev/nvme0n1p6"),
		makeUgly(nvme0n1p4, 5, "/dev/nvme0n1p5"),
	}

	correctParts := map[string]partitions{
		"/dev/sda":     sda,
		"/dev/nvme0n1": nvme0n1,
	}
	uglies := map[string]partitions{
		"/dev/sda":     uglySda,
		"/dev/nvme0n1": uglyNvme0n1,
	}

	var prettyParts = make(map[string]partitions)
	for key, uglyParts := range uglies {
		resultParts, err := redesignatePartitions(uglyParts)
		if err != nil {
			t.Log("Error rearranging partitions")
			t.Fatal(err.Error())
		}

		prettyParts[key] = resultParts
	}

	for disk, corrects := range correctParts {
		thisDiskPrettyParts := prettyParts[disk]
		for i, correct := range corrects {
			pretty := thisDiskPrettyParts[i]
			if reflect.DeepEqual(correct, pretty) {
				continue
			}

			t.Fatalf("unexpected sort result")
		}
	}
}

func makeUgly(p partition, designation int, name string) partition {
	p.designation = designation
	p.name = name
	return p
}
