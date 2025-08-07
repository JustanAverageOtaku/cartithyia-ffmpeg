package main

import (
	"cartithyia/src/feature"
	"cartithyia/src/feature/frame"
	"cartithyia/src/feature/merge"
	"flag"
	"fmt"
	"os"
)

func main() {
	featureMap := map[string]feature.Feature{
		frame.FeatureName: frame.NewFeature(flag.NewFlagSet(frame.FeatureName, flag.ExitOnError)),
		merge.FeatureName: merge.NewFeature(flag.NewFlagSet(merge.FeatureName, flag.ExitOnError)),
	}

	if len(os.Args) < 2 {
		panic("not enough arguments")
	}

	feature, ok := featureMap[os.Args[1]]
	if !ok {
		panic(fmt.Errorf("supported features:%+v, got:%s", featureMap, os.Args[1]))
	}

	if err := feature.Execute(os.Args[2:]); err != nil {
		panic(err)
	}
}
