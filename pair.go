package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"

	"gopkg.in/yaml.v1"
)

var branch = flag.String("b", "", "switch to this branch prefixed with the current pair authors")

func main() {
	flag.Usage = usage
	flag.Parse()

	configFile := os.ExpandEnv("$PAIR_GIT_CONFIG")
	if configFile == "" {
		configFile = os.ExpandEnv("$HOME/.gitconfig_local")
	}

	emailTemplate := os.ExpandEnv("$PAIR_EMAIL")
	if emailTemplate == "" {
		var err error
		emailTemplate, err = GetDefaultEmailTemplate()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: please set $PAIR_EMAIL to configure the pair email template")
			os.Exit(1)
		}
	}

	if *branch != "" {
		if switchToPairBranch(configFile, *branch, emailTemplate) {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	usernames := flag.Args()

	if len(usernames) == 0 {
		// $ pair
		if !printCurrentPairedUsers(configFile) {
			os.Exit(1)
		}
	} else {
		// $ pair author1 author2
		pairsFile := os.ExpandEnv("$PAIR_FILE")
		if pairsFile == "" {
			pairsFile = os.ExpandEnv("$HOME/.pairs")
		}

		if !setAndPrintNewPairedUsers(pairsFile, configFile, emailTemplate, usernames) {
			os.Exit(1)
		}
	}
}

func usage() {
	fmt.Println(
		`pair USER1 [USER2 [...]]
pair [OPTIONS]

Configures your git author and committer info by changing ~/.gitconfig_local.
This is meant to be used both as a means of adding multiple authors to a commit
and an alternative to editing your ~/.git_config (which is checked into git).

Options

  -b BRANCH     Switches to a git branch prefixed with the paired usernames.

Examples

  # configure paired git author info for this shell
  $ pair jsmith alice
  Alice Barns and Jon Smith <git+alice+jsmith@example.com>

  # use the same author info as the last time pair was run
  $ pair
  Alice Barns and Jon Smith <git+alice+jsmith@example.com>

  # create a branch to work on a feature
  $ pair -b ONCALL-843
  Switched to a new branch 'alice+jsmith/ONCALL-843'

Configuration

  PAIR_FILE        YAML file with a map of usernames to full names (default: ~/.pairs).
  PAIR_GIT_CONFIG  Git config file for reading and writing author info (default: ~/.gitconfig).`)

	defaultEmailTemplate, err := GetDefaultEmailTemplate()
	if err == nil {
		defaultEmailTemplate = " (default: " + defaultEmailTemplate + ")"
	}

	fmt.Println("  PAIR_EMAIL       Email address to base derived email addresses on" + defaultEmailTemplate + ".")
}

func printCurrentPairedUsers(configFile string) bool {
	name, err := gitConfig(configFile, "user.name")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to get current git author name: %v\n", err)
		return false
	}

	email, err := gitConfig(configFile, "user.email")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to get current git author email: %v\n", err)
		return false
	}

	fmt.Printf("%s <%s>\n", name, email)
	return true
}

func setAndPrintNewPairedUsers(pairsFile string, configFile string, emailTemplate string, usernames []string) bool {
	f, err := os.Open(pairsFile)
	var authorMap map[string]string
	if err == nil {
		authorMap, err = readAuthorsByUsername(bufio.NewReader(f))
	}
	if f != nil {
		f.Close()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to read authors from file (%s): %v", pairsFile, err)
		return false
	}

	sort.Strings(usernames)

	email, err := emailAddressForUsernames(emailTemplate, usernames)

	var name string

	if err == nil {
		name, err = namesForUsernames(usernames, authorMap)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return false
	}

	err = setGitConfig(configFile, "user.name", name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to set current git author name: %v\n", err)
		return false
	}

	err = setGitConfig(configFile, "user.email", email)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to set current git author name: %v\n", err)
		return false
	}

	return printCurrentPairedUsers(configFile)
}

func switchToPairBranch(configFile string, branch string, emailTemplate string) bool {
	email, err := gitConfig(configFile, "user.email")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to get current git author email from config file: %s\n", configFile)
		return false
	}

	templateUsername, _, err := SplitEmail(emailTemplate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to parse template email address: %s\n", emailTemplate)
		return false
	}

	usernames, _, err := SplitEmail(email)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to parse email address: %s\n", email)
		return false
	}

	// Remove any preceding e.g. "git+" from "git+lb+mb".
	usernames = strings.TrimPrefix(usernames, templateUsername+"+")

	fullBranch := usernames + "/" + branch

	cmd := exec.Command("git", "rev-parse", fullBranch)
	err = cmd.Run()

	args := []string{"checkout"}

	if err != nil {
		// The branch does not exist, so create it with the `-b' flag.
		args = append(args, "-b", fullBranch, "master")
	} else {
		// The branch already exists, so just switch to it.
		args = append(args, fullBranch)
	}

	cmd = exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to check out git branch: %s\n", fullBranch)
		return false
	}

	return true
}

// GetDefaultEmailTemplate determines a default email template from the current network.
func GetDefaultEmailTemplate() (string, error) {
	dnsNames, err := LookupReverseDNSNamesByInterface("en0")
	if err != nil {
		return "", err
	}

	for _, dnsName := range dnsNames {
		hostnameParts := strings.Split(dnsName, ".")
		if len(hostnameParts) >= 3 {
			return "git@" + strings.Join(hostnameParts[len(hostnameParts)-3:len(hostnameParts)-1], "."), nil
		}
	}

	return "", errors.New("expected a hostname to be a fully-qualified domain name: " + strings.Join(dnsNames, ","))
}

// LookupReverseDNSNamesByInterface finds the DNS names for the given network interface (e.g. "en0").
func LookupReverseDNSNamesByInterface(interfaceName string) ([]string, error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		cidr := addr.String()
		ip, _, err := net.ParseCIDR(cidr)
		if err == nil {
			names, err := net.LookupAddr(ip.String())
			if err == nil && len(names) > 0 {
				return names, nil
			}
		}
	}

	return nil, nil
}

// SplitEmail splits an email address into the username and the host.
// An error is returned if the email does not contain a "@" character.
func SplitEmail(email string) (string, string, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", "", errors.New("invalid email address: " + email)
	}
	return parts[0], parts[1], nil
}

// gitConfig retrieves the value of a property from a specific git config file.
// It returns the value as a string along with any error that occurred.
func gitConfig(configFile string, property string) (string, error) {
	cmd := exec.Command("git", "config", "--file", configFile, property)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimRight(string(output), "\r\n"), nil
}

// setGitConfig sets the value of a property within a specific git config file.
// It returns any error that occurred.
func setGitConfig(configFile string, property string, value string) error {
	cmd := exec.Command("git", "config", "--file", configFile, property, value)
	return cmd.Run()
}

// readAuthorsByUsername gets a map of username -> full name for possible git authors.
// pairs should be reader open to data containing a YAML map.
func readAuthorsByUsername(pairs io.Reader) (map[string]string, error) {
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

// emailAddressForUsernames generates an email address from a list of usernames.
// For example, given "michael" and "lindsay" returns "michael+lindsay".
func emailAddressForUsernames(emailTemplate string, usernames []string) (string, error) {
	user, host, err := SplitEmail(emailTemplate)
	if err != nil {
		return "", err
	}

	switch len(usernames) {
	case 0:
		return emailTemplate, nil
	case 1:
		return fmt.Sprintf("%s@%s", usernames[0], host), nil
	default:
		return fmt.Sprintf("%s+%s@%s", user, strings.Join(usernames, "+"), host), nil
	}
}

// namesForUsernames joins names corresponding to usernames with " and ".
// For example, given "michael" and "lindsay" returns "Michael Bluth and Lindsay Bluth".
func namesForUsernames(usernames []string, authorMap map[string]string) (string, error) {
	if len(usernames) == 0 {
		return "", nil
	}

	var names []string

	for _, username := range usernames {
		name, ok := authorMap[username]
		if !ok {
			return "", errors.New("no such username: " + username)
		}
		names = append(names, name)
	}

	return strings.Join(names, " and "), nil
}
