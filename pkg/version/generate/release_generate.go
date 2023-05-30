//go:build release
// +build release

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/dave/jennifer/jen"

	"github.com/weaveworks/eksctl/pkg/version"
)

const (
	versionFilename         = "pkg/version/release.go"
	defaultPreReleaseID     = "dev"
	defaultReleaseCandidate = "rc.0"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("missing argument")
	}

	command := os.Args[1]

	switch command {
	case "development":
		newVersion, newPreRelease := nextDevelopmentIteration()
		if err := writeVersionToFile(newVersion, newPreRelease, versionFilename); err != nil {
			log.Fatalf("unable to write file: %v", err)
		}

		version.Version = newVersion
		version.PreReleaseID = newPreRelease
		fmt.Println(version.GetVersion())
	case "next-pre-release-id":
		switch len(os.Args) {
		case 2:
			fmt.Println(defaultReleaseCandidate)
		case 3:
			next, err := nextPreReleaseID(os.Args[2])
			if err != nil {
				log.Fatalf("error generating next pre-release ID: %v", err)
			}
			fmt.Println(next)
		default:
			log.Fatalf("usage: release_generate %s <latest-rc-tag>", command)
		}

	case "full-version":
		fmt.Println(version.GetVersion())
	case "print-version":
		// Print simplified version X.Y.Z
		fmt.Println(version.Version)
	case "print-major-minor-version":
		fmt.Println(printMajorMinor())
	default:
		log.Fatalf("unknown option %q. Expected one of %v", command, strings.Join([]string{"development", "next-pre-release-id", "full-version", "print-version", "print-major-minor-version"}, ", "))
	}

}

func nextPreReleaseID(latestPreReleaseVersion string) (string, error) {
	if latestPreReleaseVersion == "" {
		return defaultReleaseCandidate, nil
	}

	latestPreReleaseVersion = strings.TrimPrefix(latestPreReleaseVersion, "v")
	ver, err := semver.Parse(latestPreReleaseVersion)
	if err != nil {
		return "", fmt.Errorf("invalid pre-release version: %w", err)
	}
	currentVersion, err := semver.Parse(version.Version)
	if err != nil {
		return "", fmt.Errorf("unexpected error parsing current version: %s: %w", version.Version, err)
	}

	verWithoutPre := ver
	verWithoutPre.Pre = nil
	if verWithoutPre.LT(currentVersion) || len(ver.Pre) == 0 {
		return defaultReleaseCandidate, nil
	}

	if len(ver.Pre) != 2 {
		return "", errors.New("unexpected format for PR version")
	}
	id := ver.Pre[1]
	if !id.IsNumeric() {
		return "", fmt.Errorf("expected PR version to be numeric; got %q", id.String())
	}

	return fmt.Sprintf("rc.%d", id.VersionNum+1), nil

}

func printMajorMinor() string {
	ver := semver.MustParse(version.Version)
	return fmt.Sprintf("%v.%v", ver.Major, ver.Minor)
}

func nextDevelopmentIteration() (string, string) {
	ver := semver.MustParse(version.Version)
	ver.Minor++
	return ver.String(), defaultPreReleaseID
}

func writeVersionToFile(version, preReleaseID, fileName string) error {
	f := jen.NewFilePath("pkg/version")

	f.Comment("This file was generated by release_generate.go; DO NOT EDIT.")
	f.Line()

	f.Comment("Version is the version number in semver format X.Y.Z")
	f.Var().Id("Version").Op("=").Lit(version)

	f.Comment("PreReleaseID can be empty for releases, \"rc.X\" for release candidates and \"dev\" for snapshots")
	f.Var().Id("PreReleaseID").Op("=").Lit(preReleaseID)

	f.Comment("gitCommit is the short commit hash. It will be set by the linker.")
	f.Var().Id("gitCommit").Op("=").Lit("")

	f.Comment("buildDate is the time of the build with format yyyy-mm-ddThh:mm:ssZ. It will be set by the linker.")
	f.Var().Id("buildDate").Op("=").Lit("")

	return f.Save(fileName)
}