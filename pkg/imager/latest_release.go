package imager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
)

const (
	ghOwner = "SkycoinProject"
	ghRepo  = "skybian"
)

func expectedBaseImgAssetName(tag string) string {
	return fmt.Sprintf("Skybian-%s%s", tag, ExtTarXz)
}

func LatestBaseImgURL(ctx context.Context, log logrus.FieldLogger) (string, error) {
	gh := github.NewClient(nil)
	release, _, err := gh.Repositories.GetLatestRelease(ctx, ghOwner, ghRepo)
	if err != nil {
		return "", err
	}

	tag := release.GetTagName()
	log.WithField("tag", tag).
		Debug("Got tag.")

	name := expectedBaseImgAssetName(tag)
	log.WithField("expected_name", name).
		Info("Expecting asset of name.")

	for _, asset := range release.Assets {
		if asset.GetName() != name {
			log.WithField("got", asset.GetName()).
				WithField("expected", name).
				Debug("Name does not satisfy.")
			continue
		}
		return asset.GetBrowserDownloadURL(), nil
	}
	return "", errors.New("latest release of Skybian Base Image cannot not found")
}

const TimeFormat = "2006-01-02 15:04"

type Release struct {
	Tag  string
	Type string // stable, prerelease
	Date time.Time
	URL  string
}

func (r *Release) String() string {
	return fmt.Sprintf("%s (%s) [%s]",
		r.Tag, r.Type, r.Date.Format(TimeFormat))
}

//func (r *Release) Set(s string) error {
//	buf := bytes.NewBufferString(s)
//	dateRaw, err := buf.ReadString(	']')
//	if err != nil {
//		return err
//	}
//	if r.Date, err = time.Parse(TimeFormat, strings.Trim(dateRaw, "[]")); err != nil {
//		return err
//	}
//
//	tagRaw, err := buf.ReadString(' ')
//	if err != nil {
//		return err
//	}
//	r.Tag = strings.Trim(tagRaw, " ")
//
//	typeRaw, err := buf.ReadString(')')
//	if err != nil {
//		return err
//	}
//	r.Type = strings.Trim(typeRaw, "()")
//
//	return nil
//}

func releaseURL(releases []Release, releaseStr string) (string, error) {

	tag, err := bytes.NewBufferString(releaseStr).ReadString(' ')
	if err != nil {
		return "", err
	}
	tag = strings.TrimSpace(tag)

	for _, r := range releases {
		if r.Tag != tag {
			continue
		}
		return r.URL, nil
	}

	return "", fmt.Errorf("release of tag '%s' no longer available", tag)
}

func releaseStrings(releases []Release) (rs []string) {
	rs = make([]string, len(releases))
	for i, r := range releases {
		rs[i] = r.String()
	}
	return rs
}

func ListReleases(ctx context.Context, log logrus.FieldLogger) (rs []Release, latest *Release, err error) {
	gh := github.NewClient(nil)
	ghRs, _, err := gh.Repositories.ListReleases(ctx, ghOwner, ghRepo, nil)
	if err != nil {
		return nil, nil, err
	}

	for _, r := range ghRs {
		if r.GetDraft() {
			continue
		}

		exp := expectedBaseImgAssetName(r.GetTagName())
		for _, asset := range r.Assets {
			if asset.GetName() != exp {
				continue
			}

			rInfo := Release{
				Tag:  r.GetTagName(),
				Type: "stable",
				Date: r.GetCreatedAt().Time,
				URL:  asset.GetBrowserDownloadURL(),
			}
			if r.GetPrerelease() {
				rInfo.Type = "prerelease"
			} else {
				if latest == nil || rInfo.Date.After(latest.Date) {
					latest = &rInfo
				}
			}

			log.WithField("info", rInfo).Info("Found release.")
			rs = append(rs, rInfo)
			break
		}
	}

	return rs, latest, nil
}
