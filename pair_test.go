package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
)

func TestNamesForUsernames(t *testing.T) {
	var names string
	var err error

	names, err = NamesForUsernames([]string{}, map[string]string{})
	if names != "" {
		t.Fatalf("expected empty string for empty list of usernames, got %s", names)
	}
	if err != nil {
		t.Fatalf("expected no error for empty list of usernames, got %v", err)
	}

	names, err = NamesForUsernames([]string{"mb"}, map[string]string{"mb": "Michael Bluth"})
	if names != "Michael Bluth" {
		t.Fatalf("expected 'Michael Bluth' for single username 'mb', got %s", names)
	}
	if err != nil {
		t.Fatalf("expected no error for single existing username, got %v", err)
	}

	names, err = NamesForUsernames([]string{"lb", "mb"}, map[string]string{"mb": "Michael Bluth", "lb": "Lindsay Bluth"})
	if names != "Lindsay Bluth and Michael Bluth" {
		t.Fatalf("expected 'Lindsay Bluth and Michael Bluth', got %s", names)
	}
	if err != nil {
		t.Fatalf("expected no error for two existing usernames, got %v", err)
	}

	names, err = NamesForUsernames([]string{"lb"}, map[string]string{"mb": "Michael Bluth"})
	if err == nil {
		t.Fatalf("expected error for a missing username, got nil")
	}
}

func TestEmailAddressForUsernames(t *testing.T) {
	var email string

	email = EmailAddressForUsernames([]string{})
	if email != "" {
		t.Fatalf("expected empty string for empty list of usernames, got %s", email)
	}

	email = EmailAddressForUsernames([]string{"mb"})
	if email != "mb@squareup.com" {
		t.Fatalf("expected non-paired email for a single username, got %s", email)
	}

	email = EmailAddressForUsernames([]string{"lb", "mb"})
	if email != "git+lb+mb@squareup.com" {
		t.Fatalf("expected paired email for multiple usernames, got %s", email)
	}
}

func TestReadAuthorsByUsername(t *testing.T) {
	var authorMap map[string]string
	var err error

	authorMap, err = ReadAuthorsByUsername(strings.NewReader(""))
	if len(authorMap) != 0 {
		t.Fatalf("expected reading an empty file to get zero authors, got %d", len(authorMap))
	}
	if err != nil {
		t.Fatalf("expected no error for empty authors file, got %v", err)
	}

	authorMap, err = ReadAuthorsByUsername(strings.NewReader("---\nmb: Michael Bluth"))
	if len(authorMap) != 1 || authorMap["mb"] != "Michael Bluth" {
		t.Fatalf("expected reading a single author as YAML to return one entry, got %v", authorMap)
	}
	if err != nil {
		t.Fatalf("expected reading a single author as YAML to have no errors, got %v", err)
	}

	authorMap, err = ReadAuthorsByUsername(strings.NewReader("---\nlb: Lindsay Bluth\nmb: Michael Bluth"))
	if len(authorMap) != 2 {
		t.Fatalf("expected reading multiple authors as YAML to return multiple entries, got %v", authorMap)
	}
	if err != nil {
		t.Fatalf("expected reading multiple authors as YAML to have no errors, got %v", err)
	}
}

func TestGitConfig(t *testing.T) {
	var err error
	var tempGitConfigFile *os.File

	tempGitConfigFile, err = ioutil.TempFile(os.TempDir(), "pair-git-config")
	if err != nil {
		t.Fatal("unable to create temporary git config")
	}
	tempGitConfigPath := tempGitConfigFile.Name()

	err = SetGitConfig(tempGitConfigPath, "user.name", "Michael Bluth")
	if err != nil {
		t.Fatalf("expected no error when setting git config, got %v", err)
	}

	var value string
	value, err = GitConfig(tempGitConfigPath, "user.name")
	if err != nil {
		t.Fatalf("expected no error when getting git config, got %v", err)
	}
	if value != "Michael Bluth" {
		t.Fatalf("expected getting previously-set `user.name` to have the correct value, got %s", value)
	}
}

func ExamplePrintCurrentPairedUsers() {
	var err error
	var tempGitConfigFile *os.File

	tempGitConfigFile, err = ioutil.TempFile(os.TempDir(), "pair-git-config")
	if err != nil {
		log.Fatal("unable to create temporary git config")
	}
	tempGitConfigPath := tempGitConfigFile.Name()

	err = SetGitConfig(tempGitConfigPath, "user.name", "Michael Bluth")
	if err != nil {
		log.Fatalf("expected no error when setting git config, got %v", err)
	}

	err = SetGitConfig(tempGitConfigPath, "user.email", "mb@squareup.com")
	if err != nil {
		log.Fatalf("expected no error when setting git config, got %v", err)
	}

	PrintCurrentPairedUsers(tempGitConfigPath)

	// Output:
	// Michael Bluth <mb@squareup.com>
}

func ExampleSetAndPrintNewPairedUsers() {
	var err error
	var tempPairsFile *os.File
	var tempGitConfigFile *os.File

	tempPairsFile, err = ioutil.TempFile(os.TempDir(), "pair-pairs")
	if err != nil {
		log.Fatal("unable to create temporary pairs file")
	}
	io.WriteString(tempPairsFile, "---\nmb: Michael Bluth")
	tempPairsFile.Close()

	tempGitConfigFile, err = ioutil.TempFile(os.TempDir(), "pair-git-config")
	if err != nil {
		log.Fatal("unable to create temporary git config")
	}

	SetAndPrintNewPairedUsers(tempPairsFile.Name(), tempGitConfigFile.Name(), []string{"mb"})

	var value string
	value, err = GitConfig(tempGitConfigFile.Name(), "user.name")
	if err != nil {
		log.Fatal("unable to get git config after setting users: %v", err)
	}
	fmt.Printf("user.name=%s\n", value)

	value, err = GitConfig(tempGitConfigFile.Name(), "user.email")
	if err != nil {
		log.Fatal("unable to get git config after setting users: %v", err)
	}
	fmt.Printf("user.email=%s\n", value)

	// Output:
	// Michael Bluth <mb@squareup.com>
	// user.name=Michael Bluth
	// user.email=mb@squareup.com
}
