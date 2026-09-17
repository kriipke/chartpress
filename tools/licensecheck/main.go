// SPDX-License-Identifier: Apache-2.0
// Command licensecheck enforces the repository's approved dependency licenses.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

var approvedNPM = map[string]bool{
	"0BSD": true, "Apache-2.0": true, "BlueOak-1.0.0": true,
	"BSD-2-Clause": true, "BSD-3-Clause": true, "CC0-1.0": true,
	"CC-BY-4.0": true, "ISC": true, "MIT": true, "MPL-2.0": true, "Python-2.0": true,
}

var rejectedTerms = []string{"AGPL", "BUSL", "GPL", "SSPL", "Commons-Clause"}

type module struct {
	Path string
	Dir  string
	Main bool
}

type lockfile struct {
	Packages map[string]struct {
		License any `json:"license"`
	} `json:"packages"`
}

func main() {
	var problems []string
	if err := checkGo(&problems); err != nil {
		problems = append(problems, "Go dependency inventory: "+err.Error())
	}
	if err := checkNPM(&problems); err != nil {
		problems = append(problems, "npm dependency inventory: "+err.Error())
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, p)
		}
		os.Exit(1)
	}
	fmt.Println("dependency licenses conform to policy")
}

func checkGo(problems *[]string) error {
	cmd := exec.Command("go", "list", "-mod=readonly", "-m", "-json", "all")
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var m module
		if err := dec.Decode(&m); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}
		if m.Main {
			continue
		}
		if m.Dir == "" {
			*problems = append(*problems, m.Path+": module source is not downloaded")
			continue
		}
		license, err := findLicense(m.Dir)
		if err != nil {
			*problems = append(*problems, m.Path+": "+err.Error())
			continue
		}
		if !approvedLicenseText(license) {
			*problems = append(*problems, m.Path+": unknown or unapproved license text")
		}
	}
	return nil
}

func findLicense(dir string) ([]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, prefix := range []string{"license", "copying", "unlicense", "notice"} {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if strings.HasPrefix(strings.ToLower(e.Name()), prefix) {
				return os.ReadFile(filepath.Join(dir, e.Name()))
			}
		}
	}
	return nil, errors.New("license file not found")
}

// approvedLicenseText classifies a license file by its identifying header
// rather than scanning the whole document: several copyleft licenses (e.g.
// MPL-2.0's "Secondary Licenses" clause) name GPL/AGPL within otherwise
// permissive boilerplate, which a whole-body substring scan misreads as
// copyleft.
func approvedLicenseText(b []byte) bool {
	header := string(b)
	if len(header) > 700 {
		header = header[:700]
	}
	upper := strings.ToUpper(header)
	for _, bad := range []string{
		"GNU AFFERO GENERAL PUBLIC LICENSE", "GNU GENERAL PUBLIC LICENSE",
		"GNU LESSER GENERAL PUBLIC LICENSE", "SERVER SIDE PUBLIC LICENSE",
		"BUSINESS SOURCE LICENSE", "COMMONS CLAUSE",
	} {
		if strings.Contains(upper, bad) {
			return false
		}
	}
	for _, marker := range []string{
		"Apache License", "Permission is hereby granted", "Redistribution and use in source and binary forms",
		"Mozilla Public License", "ISC License", "The Unlicense", "public domain",
		"provided 'as-is'", `provided "as-is"`, "MIT License", "MIT and Apache",
	} {
		if strings.Contains(header, marker) {
			return true
		}
	}
	return false
}

func checkNPM(problems *[]string) error {
	b, err := os.ReadFile(filepath.Join("web", "package-lock.json"))
	if err != nil {
		return err
	}
	var lock lockfile
	if err := json.Unmarshal(b, &lock); err != nil {
		return err
	}
	for path, pkg := range lock.Packages {
		if path == "" {
			continue
		}
		name := strings.TrimPrefix(path, "node_modules/")
		license, ok := pkg.License.(string)
		if !ok || strings.TrimSpace(license) == "" {
			*problems = append(*problems, name+": missing npm license metadata")
			continue
		}
		for _, bad := range rejectedTerms {
			if strings.Contains(license, bad) {
				*problems = append(*problems, name+": unapproved npm license "+license)
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		known := false
		for id := range approvedNPM {
			if license == id || strings.Contains(license, id) {
				known = true
				break
			}
		}
		if !known {
			*problems = append(*problems, name+": unknown npm license "+license)
		}
	}
	return nil
}
