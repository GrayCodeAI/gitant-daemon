package handlers

import "testing"

func TestOpenAPISpecSmartHTTPReflectsReadAndWriteGrouping(t *testing.T) {
	spec := GenerateOpenAPISpec("http://localhost:7777")

	infoRefs := pathOperation(t, spec, "/api/v1/repos/{id}/info/refs", "get")
	if summary, _ := infoRefs["summary"].(string); summary != "Advertise git smart HTTP refs" {
		t.Fatalf("unexpected info/refs summary: %q", summary)
	}
	if _, hasSecurity := infoRefs["security"]; hasSecurity {
		t.Fatalf("upload-pack info/refs should not require write security: %#v", infoRefs["security"])
	}

	uploadPack := pathOperation(t, spec, "/api/v1/repos/{id}/git-upload-pack", "post")
	if summary, _ := uploadPack["summary"].(string); summary != "Run git-upload-pack" {
		t.Fatalf("unexpected upload-pack summary: %q", summary)
	}
	if _, hasSecurity := uploadPack["security"]; hasSecurity {
		t.Fatalf("upload-pack should be documented as read access, not write-security gated: %#v", uploadPack["security"])
	}

	receivePack := pathOperation(t, spec, "/api/v1/repos/{id}/git-receive-pack", "post")
	if summary, _ := receivePack["summary"].(string); summary != "Run git-receive-pack" {
		t.Fatalf("unexpected receive-pack summary: %q", summary)
	}
	security, ok := receivePack["security"].([]map[string][]string)
	if !ok || len(security) == 0 || len(security[0]["bearerAuth"]) == 0 || security[0]["bearerAuth"][0] != "repo:write" {
		t.Fatalf("receive-pack should document write security, got %#v", receivePack["security"])
	}
}

func pathOperation(t *testing.T, spec *OpenAPISpec, path, method string) map[string]interface{} {
	t.Helper()

	pathItem, ok := spec.Paths[path].(map[string]interface{})
	if !ok {
		t.Fatalf("missing OpenAPI path %s", path)
	}
	op, ok := pathItem[method].(map[string]interface{})
	if !ok {
		t.Fatalf("missing OpenAPI operation %s %s", method, path)
	}
	return op
}
