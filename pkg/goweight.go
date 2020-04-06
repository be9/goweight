package pkg

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/mattn/go-zglob"
	"github.com/thoas/go-funk"
)

var moduleRegex = regexp.MustCompile("packagefile (.*)=(.*)")

func run(cmd []string) string {
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	os.Remove("goweight-bin-target")
	return string(out)
}

func processModule(line string) *ModuleEntry {
	captures := moduleRegex.FindAllStringSubmatch(line, -1)
	if captures == nil {
		return nil
	}
	path := captures[0][2]
	stat, _ := os.Stat(path)
	sz := uint64(stat.Size())

	return &ModuleEntry{
		Path:      path,
		Name:      captures[0][1],
		Size:      sz,
		SizeHuman: humanize.Bytes(sz),
	}
}

type ModuleEntry struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Size      uint64 `json:"size"`
	SizeHuman string `json:"size_human"`
}
type GoWeight struct {
	BuildCmd []string
}

func NewGoWeight(useTest bool) *GoWeight {
	var baseCommand string
	if useTest {
		baseCommand = "test"
	} else {
		baseCommand = "build"
	}

	cmd := []string{"go", baseCommand, "-o", "goweight-bin-target", "-work", "-a"}
	if useTest {
		cmd = append(cmd, "-c")
	}

	return &GoWeight{
		BuildCmd: cmd,
	}
}

func (g *GoWeight) BuildCurrent() string {
	d := strings.Split(strings.TrimSpace(run(g.BuildCmd)), "\n")[0]
	return strings.Split(strings.TrimSpace(d), "=")[1]
}

func (g *GoWeight) Process(work string) []*ModuleEntry {

	files, err := zglob.Glob(work + "**/importcfg")
	if err != nil {
		log.Fatal(err)
	}

	allLines := funk.Uniq(funk.FlattenDeep(funk.Map(files, func(file string) []string {
		f, err := ioutil.ReadFile(file)
		if err != nil {
			return []string{}
		}
		return strings.Split(string(f), "\n")
	})))
	modules := funk.Compact(funk.Map(allLines, processModule)).([]*ModuleEntry)
	sort.Slice(modules, func(i, j int) bool { return modules[i].Size > modules[j].Size })

	return modules
}
