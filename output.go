package main

import (
	"fmt"
	"os"
	"regexp"
)

type output interface {
	new() interface{}
	start(ctx chan Context) error
	configure(f map[string]string) error
}

var output_plugins = make(map[string]output)

func RegisterOutput(name string, out output) {
	if out == nil {
		Log("output: Register output is nil")
	}

	if _, ok := output_plugins[name]; ok {
		Log("output: Register called twice for output " + name)
	}

	output_plugins[name] = out
}

func NewOutputs(ctx chan Context) error {
	outChan := make([]chan Context, 0)
	for _, output_config := range config.Outputs_config {
		f := output_config.(map[string]string)
		tmpch := make(chan Context)
		outChan = append(outChan, tmpch)
		go func(f map[string]string, tmpch chan Context) {
			output_type, ok := f["type"]
			if !ok {
				fmt.Println("no type configured")
				os.Exit(-1)
			}

			output_plugin, ok := output_plugins[output_type]
			if !ok {
				Log("unkown type ", output_type)
				os.Exit(-1)
			}

			out := output_plugin.new()
			err := out.(output).configure(f)
			if err != nil {
				Log(err)
			}

			err = out.(output).start(tmpch)
			if err != nil {
				Log(err)
			}
		}(f, tmpch)
	}

	for {
		s := <-ctx
		for i, o := range outChan {
			f := config.Outputs_config[i].(map[string]string)
			tag := f["tag"]

			chunk, err := BuildRegexpFromGlobPattern(tag)
			if err != nil {
				return err
			}

			re, err := regexp.Compile(chunk)
			if err != nil {
				return err
			}

			flag := re.MatchString(s.tag)
			if flag == false {
				continue
			}

			o <- s
		}
	}

	return nil
}
