package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/serejja/grpcurl"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"io/ioutil"
	"net/http"
	"time"
)

type GRPCRequest struct {
	Address    string          `json:"address"`
	Method     string          `json:"method"`
	Proto      string          `json:"proto"`
	ImportPath string          `json:"import_path"`
	Data       json.RawMessage `json:"data"`
}

var (
	httpPort = flag.Int("port", 9000, "HTTP bind port")
)

func main() {
	flag.Parse()

	http.HandleFunc("/rpc", HandleRPC)
	http.HandleFunc("/list", HandleList)
	http.HandleFunc("/describe", HandleDescribe)

	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", *httpPort), nil)
}

func HandleRPC(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	request := new(GRPCRequest)
	err = json.Unmarshal(body, &request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	ctx := context.Background()
	cc, err := dial(request.Address, ctx)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	in := bytes.NewReader(request.Data)
	descSource, err := descriptorSource(cc, ctx, request.Proto, request.ImportPath)
	rf, formatter, err := grpcurl.RequestParserAndFormatterFor(grpcurl.Format("json"), descSource, true, true, in)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	h := grpcurl.NewDefaultEventHandler(w, descSource, formatter, false)
	err = grpcurl.InvokeRPC(ctx, descSource, cc, request.Method, []string{}, h, rf.Next)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
}

func HandleList(w http.ResponseWriter, r *http.Request) {
	// TODO implement

	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func HandleDescribe(w http.ResponseWriter, r *http.Request) {
	// TODO implement

	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func dial(target string, ctx context.Context) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return grpcurl.BlockingDial(ctx, "tcp", target, nil)
}

func descriptorSource(cc *grpc.ClientConn, ctx context.Context, protoFile string, importPath string) (grpcurl.DescriptorSource, error) {
	if protoFile != "" {
		return grpcurl.DescriptorSourceFromProtoFiles([]string{importPath}, protoFile)
	} else {
		md := grpcurl.MetadataFromHeaders([]string{})
		refCtx := metadata.NewOutgoingContext(ctx, md)
		refClient := grpcreflect.NewClient(refCtx, reflectpb.NewServerReflectionClient(cc))
		defer refClient.Reset()
		return grpcurl.DescriptorSourceFromServer(ctx, refClient), nil
	}
}
