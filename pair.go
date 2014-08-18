package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"

	"gopkg.in/yaml.v1"
)

func init() {
	flag.Usage = func() {
		fmt.Println("pair USER1 [USER2 [...]]")
		fmt.Println("pair [OPTIONS]")
		fmt.Println("")
		fmt.Println("Configures your git author and committer info by changing ~/.gitconfig_local.")
		fmt.Println("This is meant to be used both as a means of adding multiple authors to a commit")
		fmt.Println("and an alternative to editing your ~/.git_config (which is checked into git).")
		fmt.Println("")
		fmt.Println("Examples")
		fmt.Println("")
		fmt.Println("  # configure paired git author info for this shell")
		fmt.Println("  $ pair jsmith alice")
		fmt.Println("  Alice Barns and Jon Smith <git+alice+jsmith@squareup.com>")
		fmt.Println("")
		fmt.Println("  # use the same author info as the last time pair was run")
		fmt.Println("  $ pair")
		fmt.Println("  Alice Barns and Jon Smith <git+alice+jsmith@squareup.com>")
	}
}

func main() {
	flag.Parse()

	configFile := os.ExpandEnv("$HOME/.gitconfig_local")
	usernames := flag.Args()

	if len(usernames) == 0 {
		// $ pair
		if !PrintCurrentPairedUsers(configFile) {
			os.Exit(1)
		}
	} else {
		// $ pair author1 author2
		pairsFile := os.ExpandEnv("$PAIR_FILE")
		if pairsFile == "" {
			pairsFile = os.ExpandEnv("$HOME/.pairs")
		}

		if !SetAndPrintNewPairedUsers(pairsFile, configFile, usernames) {
			os.Exit(1)
		}
	}
}

func PrintCurrentPairedUsers(configFile string) bool {
	var err error
	var name string
	var email string

	name, err = GitConfig(configFile, "user.name")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to get current git author name: %v\n", err)
		return false
	}

	email, err = GitConfig(configFile, "user.email")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to get current git author email: %v\n", err)
		return false
	}

	fmt.Printf("%s <%s>\n", name, email)
	return true
}

func SetAndPrintNewPairedUsers(pairsFile string, configFile string, usernames []string) bool {
	var err error
	var name string
	var email string

	f, err := os.Open(pairsFile)
	var authorMap map[string]string
	if err == nil {
		authorMap, err = ReadAuthorsByUsername(bufio.NewReader(f))
	}
	if f != nil {
		f.Close()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to read authors from file (%s): %v", pairsFile, err)
		return false
	}

	sort.Strings(usernames)

	email = EmailAddressForUsernames(usernames)
	name, err = NamesForUsernames(usernames, authorMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return false
	}

	err = SetGitConfig(configFile, "user.name", name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to set current git author name: %v\n", err)
		return false
	}

	err = SetGitConfig(configFile, "user.email", email)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to set current git author name: %v\n", err)
		return false
	}

	return PrintCurrentPairedUsers(configFile)
}

// GitConfig retrieves the value of a property from a specific git config file.
// It returns the value as a string along with any error that occurred.
func GitConfig(configFile string, property string) (string, error) {
	cmd := exec.Command("git", "config", "--file", configFile, property)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimRight(string(output), "\r\n"), nil
}

// SetGitConfig sets the value of a property within a specific git config file.
// It returns any error that occurred.
func SetGitConfig(configFile string, property string, value string) error {
	cmd := exec.Command("git", "config", "--file", configFile, property, value)
	return cmd.Run()
}

// ReadAuthorsByUsername gets a map of username -> full name for possible git authors.
// pairsFile should be a path to a file containing a YAML map.
func ReadAuthorsByUsername(pairs io.Reader) (map[string]string, error) {
	var authorMap map[string]string

	bytes, err := ioutil.ReadAll(pairs)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, &authorMap)
	if err != nil {
		return nil, err
	}

	return authorMap, nil
}

// EmailAddressForUsernames generates an email address from a list of usernames.
// For example, given "michael" and "lindsay" returns "michael+lindsay".
func EmailAddressForUsernames(usernames []string) string {
	switch len(usernames) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%s@squareup.com", usernames[0])
	default:
		return fmt.Sprintf("git+%s@squareup.com", strings.Join(usernames, "+"))
	}
}

// NamesForUsernames joins names corresponding to usernames with " and ".
// For example, given "michael" and "lindsay" returns "Michael Bluth and Lindsay Bluth".
func NamesForUsernames(usernames []string, authorMap map[string]string) (string, error) {
	if len(usernames) == 0 {
		return "", nil
	}

	names := make([]string, 0, 0)

	for _, username := range usernames {
		name, ok := authorMap[username]
		if !ok {
			return "", errors.New("no such username: " + username)
		}
		names = append(names, name)
	}

	return strings.Join(names, " and "), nil
}
