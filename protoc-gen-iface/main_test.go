package main

import (
	"io"
	"log"
	"os"
	"sync"
	"testing"

	goproto "github.com/golang/protobuf/proto"
	"github.com/josudoey/pbconv"
	"github.com/josudoey/pbconv/internal/fixture"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestMain(t *testing.T) {
	testStdoutReader, testStdout, err := os.Pipe()
	if err != nil {
		t.Fatalf("os pipe got err: %+v\n", err)
	}

	stdoutChan := make(chan []byte, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		originStdout := os.Stdout
		os.Stdout = testStdout
		defer func() {
			os.Stdout = originStdout
		}()

		wg.Done()
		responseContent, _ := io.ReadAll(testStdoutReader)
		stdoutChan <- responseContent
		close(stdoutChan)
	}()

	testStdin, testStdinWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os pipe got err: %+v\n", err)
	}

	originArgs := os.Args
	os.Args = os.Args[:1]
	defer func() {
		os.Args = originArgs
	}()

	originStdin := os.Stdin
	os.Stdin = testStdin
	defer func() {
		os.Stdin = originStdin
	}()

	fileProtoPath := fixture.File_internal_fixture_file_proto.Path()
	protoFile, _ := pbconv.GetFileDescriptorProtoByFilename(fileProtoPath)
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{fileProtoPath},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			protoFile,
		},
	}

	requestContent, err := goproto.Marshal(req)
	if err != nil {
		t.Fatalf("proto Marshal got err: %+v\n", err)
	}
	testStdinWriter.Write(requestContent)
	testStdinWriter.Close()

	wg.Wait()
	main()
	testStdout.Close()

	requestContent = <-stdoutChan
	res := pluginpb.CodeGeneratorResponse{}
	err = goproto.Unmarshal(requestContent, &res)
	if err != nil {
		t.Fatalf("response Unmarshal got err: %+v", err)
	}

	if want := 1; len(res.File) != want {
		log.Fatalf("response file length got: %q want: %q", len(res.File), want)
	}

	if want := "github.com/josudoey/pbconv/internal/fixture/file.iface.go"; want != res.File[0].GetName() {
		t.Errorf("response file name got: %q want: %q", res.File[0].GetName(), want)
	}

	if want := `// Code generated by protoc-gen-iface. DO NOT EDIT.
package fixture
`; want != res.File[0].GetContent() {
		t.Errorf("response file content got: %q want: %q", res.File[0].GetContent(), want)
	}
}
