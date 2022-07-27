# sfdisk-sort-go

sfdisk-sort is a small text processing Go script for `sfdisk(8)` dump output (e.g. from `# sfdisk -d /dev/sdX` command). The fact that it has suffix `-go` is because I intend to write another implementation of this program in Rust (`sfdisk-sort-rs`).

## Why

If you partition your disks a lot, then you would likely encountered a situation where you ended up with ugly partition names when sorted by start blocks (e.g. with `lsblk` output).

I usually find myself end up whining about this little inconvenience, and sometimes the fact that if I edit some of my partitions - my perfectly beautiful partition table will be fucked up and I will have to endure [this process](https://unix.stackexchange.com/questions/18752/change-the-number-of-the-partition-from-sda1-to-sda2) again.

So I wrote this script based one particular solution from [unix.stackexchange.com](https://unix.stackexchange.com/questions/18752/change-the-number-of-the-partition-from-sda1-to-sda2). The forum answer involves using `sfdisk(8)` to dump the disk partition table into a text file, and then have the user manually edit that partition info text, and then use `sfdisk(8)` to _throw it back to partition table_.

And as someone who regularly fucks with my disk partitions (like destroying/adding/resizing them, BUT not the actual fucking), I decide to write this small Go script to parse `sfdisk(8)` output and spit out the one with the beautifully arranged partition numbers.

In short, this script _does not_ modify your partition table, but instead, it spits out a hopefully useful text for `sfdisk(8)` to read back and apply it to partition table.

> sfdisk-sort-helper is tested for `/dev/sdX` (which implies that it should also works with `/dev/vdX` and other similar names), and `/dev/nvmeXnYpZ` schemes.


## `sfdisk-sort-go` Requirements:

- The only dependencies of this script is the [Go progamming language](https://golang.org).

- [`sfdisk`](https://en.wikipedia.org/wiki/Sfdisk)

- Root permission - it will be needed to run [`sfdisk`](https://en.wikipedia.org/wiki/Sfdisk) in the first place.

## `sfdisk-sort-go` Usage example

There are 2 ways `sfdisk-sort` gets its input from. See usage examples below

### (1) `sfdisk-sort` will call `sfdisk -d` on its own argument:

```
$ # Compile from source:
$
$ go build ./cmd/sfdisk-sort;
$
$ # Generate the sorted sfdisk partition dump text,
$ # and write the output to sda.parttab.bkp.
$ # The output also contains original sfdisk dump text,
$ # although it is commented out with a "#".
$ # Also note that you'll need sudo for sfdisk to be able
$ # to read the Linux kernel partition table
$
$ sudo ./sfdisk-sort /dev/sdb > sdb.parttab.bkp;
$
$ # Read this file back to partition table.
$ # DO THIS AT YOUR OWN DISK.
$
$ sudo sfdisk /dev/sdb < sdb.parttab.bkp;
$
$ # If the above command does not work, try:
$
$ sudo sfdisk --no-reread -f /dev/sdb < sdb.new;
```

### (2) Pipe `sfdisk -d` output to `sfdisk-sort`:

```
$ # Compile from source:
$
$ go build ./cmd/sfdisk-sort;
$
$ # Pipe `sfdisk -d <disk>` command into sfdisk-sort,
$ # and redirect the script's output to sda.parttab.bkp.
$ # The output also contains original sfdisk dump text,
$ # although it is commented out with a "#".
$ # Also note that you'll need sudo for sfdisk to be able
$ # to read the Linux kernel partition table
$
$ sudo sfdisk -d /dev/sdb | ./sfdisk-sort -stdin > sdb.parttab.bkp;
$
$ # Read this file back to partition table.
$ # DO THIS AT YOUR OWN DISK.
$
$ sudo sfdisk /dev/sdb < sdb.parttab.bkp;
$
$ # If the above command does not work, try:
$
$ sudo sfdisk --no-reread -f /dev/sdb < sdb.new;
```
