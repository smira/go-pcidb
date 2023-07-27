// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build exclude

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Source data: pci.ids
// Source can be updated by downloading new version from https://pci-ids.ucw.cz/.

// Some code in this file is based on https://github.com/jaypipes/pcidb.

func main() {
	log.SetFlags(0)
	log.SetPrefix("pcidb-gen: ")

	var g Generator

	cfg := &packages.Config{
		Mode:  packages.LoadSyntax,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		log.Fatal(err)
	}
	if len(pkgs) != 1 {
		log.Fatalf("error: %d packages found", len(pkgs))
	}

	pkg := pkgs[0]

	// Print the header and package clause.

	g.Printf("// This Source Code Form is subject to the terms of the Mozilla Public\n")
	g.Printf("// License, v. 2.0. If a copy of the MPL was not distributed with this\n")
	g.Printf("// file, You can obtain one at http://mozilla.org/MPL/2.0/.\n\n")

	g.Printf("// Code generated by \"pcidb-gen %s\"; DO NOT EDIT.\n", strings.Join(os.Args[1:], " "))
	g.Printf("\n")
	g.Printf("package %s\n", pkg.Name)

	if err = parseDBFile(); err != nil {
		log.Fatalf("error: %s", err)
	}

	g.Printf("func lookupClass(key Class) (string, bool) {\n")
	g.Printf("\tswitch (key) {\n")

	classIDs := make([]Class, 0, len(Classes))

	for classID := range Classes {
		classIDs = append(classIDs, classID)
	}

	sort.Slice(classIDs, func(i, j int) bool { return classIDs[i] < classIDs[j] })

	for _, classID := range classIDs {
		g.Printf("\t\tcase 0x%02x: return %q, true\n", classID, Classes[classID])
	}

	g.Printf("\tdefault: return \"\", false\n")
	g.Printf("\t}\n")

	g.Printf("}\n")

	g.Printf("\nfunc lookupSubclass(key ClassSubclass) (string, bool) {\n")
	g.Printf("\tswitch (key) {\n")

	subclassIDs := make([]ClassSubclass, 0, len(Subclasses))

	for subclassID := range Subclasses {
		subclassIDs = append(subclassIDs, subclassID)
	}

	sort.Slice(subclassIDs, func(i, j int) bool { return subclassIDs[i] < subclassIDs[j] })

	for _, subclassID := range subclassIDs {
		g.Printf("case 0x%04x: return %q, true\n", subclassID, Subclasses[subclassID])
	}

	g.Printf("\tdefault: return \"\", false\n")
	g.Printf("\t}\n")

	g.Printf("}\n")

	g.Printf("\nfunc lookupProgrammingInterface(key ClassSubclassProgrammingInterface) (string, bool) {\n")
	g.Printf("\tswitch (key) {\n")

	piIDs := make([]ClassSubclassProgrammingInterface, 0, len(ProgrammingInterfaces))

	for piID := range ProgrammingInterfaces {
		piIDs = append(piIDs, piID)
	}

	sort.Slice(piIDs, func(i, j int) bool { return piIDs[i] < piIDs[j] })

	for _, piID := range piIDs {
		g.Printf("case 0x%06x: return %q, true\n", piID, ProgrammingInterfaces[piID])
	}

	g.Printf("\tdefault: return \"\", false\n")
	g.Printf("\t}\n")

	g.Printf("}\n")

	g.Printf("\nfunc lookupVendor(key Vendor) (string, bool) {\n")
	g.Printf("\tswitch (key) {\n")

	vendorIDs := make([]Vendor, 0, len(Vendors))

	for vendorID := range Vendors {
		vendorIDs = append(vendorIDs, vendorID)
	}

	sort.Slice(vendorIDs, func(i, j int) bool { return vendorIDs[i] < vendorIDs[j] })

	for _, vendorID := range vendorIDs {
		g.Printf("case 0x%04x: return %q, true \n", vendorID, Vendors[vendorID])
	}

	g.Printf("\tdefault: return \"\", false\n")
	g.Printf("\t}\n")

	g.Printf("}\n")

	g.Printf("\nfunc lookupProduct(key VendorProduct) (string, bool) {\n")
	g.Printf("\tswitch (key) {\n")

	productIDs := make([]VendorProduct, 0, len(Products))

	for productID := range Products {
		productIDs = append(productIDs, productID)
	}

	sort.Slice(productIDs, func(i, j int) bool { return productIDs[i] < productIDs[j] })

	for _, productID := range productIDs {
		g.Printf("case 0x%08x: return %q, true\n", productID, Products[productID])
	}

	g.Printf("\tdefault: return \"\", false\n")
	g.Printf("\t}\n")

	g.Printf("}\n")

	g.Printf("\nfunc lookupSubsystem(key VendorProductSubsystem) (SubsystemInfo, bool) {\n")
	g.Printf("\tswitch (key) {\n")

	subsystemIDs := make([]VendorProductSubsystem, 0, len(Subsystems))

	for subsystemID := range Subsystems {
		subsystemIDs = append(subsystemIDs, subsystemID)
	}

	sort.Slice(subsystemIDs, func(i, j int) bool { return subsystemIDs[i] < subsystemIDs[j] })

	for _, subsystemID := range subsystemIDs {
		g.Printf("case 0x%012x: return SubsystemInfo{ Vendor: 0x%04x, Name: %q }, true\n", subsystemID, Subsystems[subsystemID].Vendor, Subsystems[subsystemID].Name)
	}

	g.Printf("\tdefault: return SubsystemInfo{}, false\n")
	g.Printf("\t}\n")

	g.Printf("}\n")

	src := g.format()

	if err := os.WriteFile("db.go", src, 0o644); err != nil {
		log.Fatalf("writing output: %s", err)
	}
}

type Generator struct {
	buf bytes.Buffer
}

func (g *Generator) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

// format returns the gofmt-ed contents of the Generator's buffer.
func (g *Generator) format() []byte {
	src, err := format.Source(g.buf.Bytes())
	if err != nil {
		// Should never happen, but can arise when developing this code.
		// The user can compile the output to see the error.
		log.Printf("warning: internal error: invalid Go generated: %s", err)
		log.Printf("warning: compile the package to analyze the error")
		return g.buf.Bytes()
	}

	return src
}

type (
	Class                             = uint8
	ClassSubclass                     = uint16 // Class + Subclass
	ClassSubclassProgrammingInterface = uint32 // Class + Subclass + ProgrammingInterface
	Vendor                            = uint16
	VendorProduct                     = uint32 // Vendor + Product
	VendorProductSubsystem            = uint64 // Vendor + Product + Subsystem
	Subsystem                         struct {
		Vendor Vendor
		Name   string
	}
)

var (
	Classes               = make(map[Class]string)
	Vendors               = make(map[Vendor]string)
	Subclasses            = make(map[ClassSubclass]string)
	Products              = make(map[VendorProduct]string)
	ProgrammingInterfaces = make(map[ClassSubclassProgrammingInterface]string)
	Subsystems            = make(map[VendorProductSubsystem]Subsystem)
)

func mustParse8(in []rune) uint8 {
	v, err := strconv.ParseUint(string(in), 16, 8)
	if err != nil {
		panic(err)
	}

	return uint8(v)
}

func mustParse16(in []rune) uint16 {
	v, err := strconv.ParseUint(string(in), 16, 16)
	if err != nil {
		panic(err)
	}

	return uint16(v)
}

func parseDBFile() error {
	in, err := os.Open("pci.ids")
	if err != nil {
		return err
	}

	defer in.Close()

	scanner := bufio.NewScanner(in)

	inClassBlock := false

	var (
		curClass         Class
		curClassSubclass ClassSubclass
		curVendor        Vendor
		curVendorProduct VendorProduct
	)

	for scanner.Scan() {
		line := scanner.Text()
		// skip comments and blank lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		lineBytes := []rune(line)

		// Lines starting with an uppercase "C" indicate a PCI top-level class
		// information block. These lines look like this:
		//
		// C 02  Network controller
		if lineBytes[0] == 'C' {
			inClassBlock = true

			curClass = mustParse8(lineBytes[2:4])
			className := string(lineBytes[6:])

			Classes[curClass] = className

			continue
		}

		// Lines not beginning with an uppercase "C" or a TAB character
		// indicate a top-level vendor information block. These lines look like
		// this:
		//
		// 0a89  BREA Technologies Inc
		if lineBytes[0] != '\t' {
			inClassBlock = false

			curVendor = mustParse16(lineBytes[0:4])
			vendorName := string(lineBytes[6:])

			Vendors[curVendor] = vendorName

			continue
		}

		// Lines beginning with only a single TAB character are *either* a
		// subclass OR are a device information block. If we're in a class
		// block (i.e. the last parsed block header was for a PCI class), then
		// we parse a subclass block. Otherwise, we parse a device dbrmation
		// block.
		//
		// A subclass information block looks like this:
		//
		// \t00  Non-VGA unclassified device
		//
		// A device information block looks like this:
		//
		// \t0002  PCI to MCA Bridge
		if len(lineBytes) > 1 && lineBytes[1] != '\t' {
			if inClassBlock {
				subclassID := mustParse8(lineBytes[1:3])
				curClassSubclass = ClassSubclass(uint16(curClass)<<8 | uint16(subclassID))
				subclassName := string(lineBytes[5:])

				Subclasses[curClassSubclass] = subclassName
			} else {
				productID := mustParse16(lineBytes[1:5])
				productName := string(lineBytes[7:])

				curVendorProduct = VendorProduct(uint32(curVendor)<<16 | uint32(productID))

				Products[curVendorProduct] = productName
			}
		} else {
			// Lines beginning with two TAB characters are *either* a subsystem
			// (subdevice) OR are a programming interface for a PCI device
			// subclass. If we're in a class block (i.e. the last parsed block
			// header was for a PCI class), then we parse a programming
			// interface block, otherwise we parse a subsystem block.
			//
			// A programming interface block looks like this:
			//
			// \t\t00  UHCI
			//
			// A subsystem block looks like this:
			//
			// \t\t0e11 4091  Smart Array 6i
			if inClassBlock {
				progIfaceID := mustParse8(lineBytes[2:4])
				progIfaceName := string(lineBytes[6:])

				ProgrammingInterfaces[ClassSubclassProgrammingInterface(uint32(curClassSubclass)<<8|uint32(progIfaceID))] = progIfaceName
			} else {
				vendorID := mustParse16(lineBytes[2:6])
				subsystemID := mustParse16(lineBytes[7:11])
				subsystemName := string(lineBytes[13:])

				Subsystems[VendorProductSubsystem(uint64(curVendorProduct)<<16|uint64(subsystemID))] = Subsystem{
					Vendor: vendorID,
					Name:   subsystemName,
				}
			}
		}
	}

	return nil
}
