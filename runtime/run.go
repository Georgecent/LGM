package runtime

import (
	"fmt"
	"github.com/logrusorgru/aurora"
)

func title(s string) string {
	return aurora.Bold(s).String()
}

func Run(options Options) {
	doExport := options.ExportFile != ""

	// Todo
	//doBuild := len(options.BuildArgs) > 0
	//isCi, _ := strconv.ParseBool(os.Getenv("CI"))

	// Todo
	//if doBuild {
	//	fmt.Println(title("Buliding image..."))
	//	options.ImageId = runBuild(options.BuildArgs)
	//}

	//对于一个已存在的镜像
	//Fetching image... (this can take a while with large images)
	//Parsing image...
	//Analyzing image...
	//Building cache...

	fmt.Println(title("Fetching image...") + " (this can take a while with large images)")
	// Todo fetching
	//

	fmt.Println(title("Parsing image..."))
	// Todo Parsing


	if doExport {
		fmt.Println(title(fmt.Sprintf("Analyzing image... (export to '%s')", options.ExportFile)))
	} else {
		fmt.Println(title("Analyzing image..."))
	}

}


