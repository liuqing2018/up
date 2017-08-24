package open

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"

	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"github.com/tj/go/term"
	"github.com/tj/kingpin"

	"github.com/apex/up/internal/cli/root"
	"github.com/apex/up/internal/colors"
	"github.com/apex/up/internal/stats"
	"github.com/apex/up/internal/util"
)

func init() {
	cmd := root.Command("build", "Build zip file.")
	size := cmd.Flag("size", "Show zip contents size information.").Bool()
	cmd.Example(`up build`, "Build archive and save to ./out.zip")
	cmd.Example(`up build > /tmp/out.zip`, "Build archive and output to file via stdout.")
	cmd.Example(`up build --size`, "Build archive list files by size.")
	cmd.Example(`up build --size > /dev/null`, "Build archive and list size without creating out.zip.")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		defer util.Pad()()

		_, p, err := root.Init()
		if err != nil {
			return errors.Wrap(err, "initializing")
		}

		stats.Track("Build", nil)

		if err := p.Build(); err != nil {
			return errors.Wrap(err, "building")
		}

		r, err := p.Zip()
		if err != nil {
			return errors.Wrap(err, "zip")
		}

		out := os.Stdout

		if term.IsTerminal() {
			f, err := os.Create("out.zip")
			if err != nil {
				return errors.Wrap(err, "creating zip")
			}
			defer f.Close()
			out = f
		}

		var buf bytes.Buffer
		if *size {
			r = io.TeeReader(r, &buf)
		}

		if _, err := io.Copy(out, r); err != nil {
			return errors.Wrap(err, "copying")
		}

		if *size {
			var files []*zip.File

			z, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			if err != nil {
				return errors.Wrap(err, "opening zip")
			}

			for _, f := range z.File {
				files = append(files, f)
			}

			sort.Slice(files, func(i int, j int) bool {
				a := files[i]
				b := files[j]
				return a.UncompressedSize64 > b.UncompressedSize64
			})

			fmt.Fprintln(os.Stderr)
			for i, f := range files {
				// skip out.zip
				if i == 0 {
					continue
				}

				size := humanize.Bytes(f.UncompressedSize64)
				fmt.Fprintf(os.Stderr, "  %10s %s\n", size, colors.Purple(f.Name))
			}
		}

		return err
	})
}
