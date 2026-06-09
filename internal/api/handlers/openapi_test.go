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

func TestOpenAPISpecIncludesCollaboratorManagementRoutes(t *testing.T) {
	spec := GenerateOpenAPISpec("http://localhost:7777")

	list := pathOperation(t, spec, "/api/v1/repos/{id}/collaborators", "get")
	if summary, _ := list["summary"].(string); summary != "List repository collaborators" {
		t.Fatalf("unexpected collaborator list summary: %q", summary)
	}

	add := pathOperation(t, spec, "/api/v1/repos/{id}/collaborators", "post")
	if summary, _ := add["summary"].(string); summary != "Add repository collaborator" {
		t.Fatalf("unexpected collaborator add summary: %q", summary)
	}
	assertRepoWriteSecurity(t, add)

	remove := pathOperation(t, spec, "/api/v1/repos/{id}/collaborators/{user}", "delete")
	if summary, _ := remove["summary"].(string); summary != "Remove repository collaborator" {
		t.Fatalf("unexpected collaborator remove summary: %q", summary)
	}
	assertRepoWriteSecurity(t, remove)
}

func TestOpenAPISpecIncludesReviewCommentRoutes(t *testing.T) {
	spec := GenerateOpenAPISpec("http://localhost:7777")

	list := pathOperation(t, spec, "/api/v1/repos/{id}/prs/{prId}/review", "get")
	if summary, _ := list["summary"].(string); summary != "List PR review comments" {
		t.Fatalf("unexpected review comment list summary: %q", summary)
	}

	create := pathOperation(t, spec, "/api/v1/repos/{id}/prs/{prId}/review", "post")
	if summary, _ := create["summary"].(string); summary != "Create PR review comment" {
		t.Fatalf("unexpected review comment create summary: %q", summary)
	}
	assertRepoWriteSecurity(t, create)

	resolve := pathOperation(t, spec, "/api/v1/review-comments/{commentId}/resolve", "post")
	if summary, _ := resolve["summary"].(string); summary != "Resolve review comment" {
		t.Fatalf("unexpected review comment resolve summary: %q", summary)
	}
	assertBearerSecurity(t, resolve)

	remove := pathOperation(t, spec, "/api/v1/review-comments/{commentId}", "delete")
	if summary, _ := remove["summary"].(string); summary != "Delete review comment" {
		t.Fatalf("unexpected review comment delete summary: %q", summary)
	}
	assertBearerSecurity(t, remove)
}

func assertRepoWriteSecurity(t *testing.T, op map[string]interface{}) {
	t.Helper()
	security, ok := op["security"].([]map[string][]string)
	if !ok || len(security) == 0 || len(security[0]["bearerAuth"]) == 0 || security[0]["bearerAuth"][0] != "repo:write" {
		t.Fatalf("operation should document write security, got %#v", op["security"])
	}
}

func assertBearerSecurity(t *testing.T, op map[string]interface{}) {
	t.Helper()
	security, ok := op["security"].([]map[string][]string)
	if !ok || len(security) == 0 {
		t.Fatalf("operation should document bearer security, got %#v", op["security"])
	}
	if _, ok := security[0]["bearerAuth"]; !ok {
		t.Fatalf("operation should document bearerAuth security, got %#v", op["security"])
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
