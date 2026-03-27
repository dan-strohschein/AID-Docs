package validator

import (
	"testing"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

func TestHeaderComplete(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1
`)
	issues := Validate(f)
	for _, i := range issues {
		if i.Rule == "header-complete" {
			t.Errorf("unexpected header-complete issue: %s", i.Message)
		}
	}
}

func TestHeaderMissingModule(t *testing.T) {
	f, _, _ := parser.ParseString(`@lang go
@version 1.0.0
@aid_version 0.1
`)
	issues := Validate(f)
	found := false
	for _, i := range issues {
		if i.Rule == "header-complete" && i.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Error("expected header-complete error for missing @module")
	}
}

func TestRequiredFieldsFn(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test
@lang go
@version 1.0.0
@aid_version 0.1

---

@fn Get
@purpose Get a value
@sig (key: str) -> str
`)
	issues := Validate(f)
	for _, i := range issues {
		if i.Rule == "required-fields" && i.Entry == "Get" {
			t.Errorf("unexpected required-fields issue on Get: %s", i.Message)
		}
	}
}

func TestRequiredFieldsFnMissingSig(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test
@lang go
@version 1.0.0
@aid_version 0.1

---

@fn Get
@purpose Get a value
`)
	issues := Validate(f)
	found := false
	for _, i := range issues {
		if i.Rule == "required-fields" && i.Entry == "Get" && i.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Error("expected required-fields error for missing @sig on @fn")
	}
}

func TestMethodBinding(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test
@lang go
@version 1.0.0
@aid_version 0.1

---

@type Config
@kind struct
@purpose Config
@methods Validate

---

@fn Config.Validate
@purpose Validate config
@sig (self) -> error
`)
	issues := Validate(f)
	for _, i := range issues {
		if i.Rule == "method-binding" {
			t.Errorf("unexpected method-binding issue: %s", i)
		}
	}
}

func TestMethodBindingMissingType(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test
@lang go
@version 1.0.0
@aid_version 0.1

---

@fn Missing.Validate
@purpose Validate
@sig (self) -> error
`)
	issues := Validate(f)
	found := false
	for _, i := range issues {
		if i.Rule == "method-binding" && i.Entry == "Missing.Validate" {
			found = true
		}
	}
	if !found {
		t.Error("expected method-binding warning for Missing.Validate")
	}
}

func TestMethodBindingNotInMethodsList(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test
@lang go
@version 1.0.0
@aid_version 0.1

---

@type Config
@kind struct
@purpose Config
@methods Other

---

@fn Config.Validate
@purpose Validate
@sig (self) -> error
`)
	issues := Validate(f)
	found := false
	for _, i := range issues {
		if i.Rule == "method-binding" && i.Entry == "Config.Validate" && i.Severity == SeverityInfo {
			found = true
		}
	}
	if !found {
		t.Error("expected method-binding info for Config.Validate not in @methods")
	}
}

func TestCrossReferences(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test
@lang go
@version 1.0.0
@aid_version 0.1

---

@fn Get
@purpose Get
@sig (key: str) -> str
@related Set, NonExistent
`)
	issues := Validate(f)
	found := false
	for _, i := range issues {
		if i.Rule == "cross-references" && i.Entry == "Get" {
			found = true
		}
	}
	if !found {
		t.Error("expected cross-references issue for NonExistent")
	}
}

func TestStatusValid(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test
@lang go
@version 1.0.0
@aid_status reviewed
@aid_version 0.1
`)
	issues := Validate(f)
	for _, i := range issues {
		if i.Rule == "status-valid" {
			t.Errorf("unexpected status-valid issue: %s", i.Message)
		}
	}
}

func TestStatusInvalid(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test
@lang go
@version 1.0.0
@aid_status bogus
@aid_version 0.1
`)
	issues := Validate(f)
	found := false
	for _, i := range issues {
		if i.Rule == "status-valid" {
			found = true
		}
	}
	if !found {
		t.Error("expected status-valid warning for bogus status")
	}
}

func TestCodeVersionValid(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test
@lang go
@version 1.0.0
@code_version git:979fe97
@aid_version 0.1
`)
	issues := Validate(f)
	for _, i := range issues {
		if i.Rule == "code-version-format" {
			t.Errorf("unexpected code-version issue: %s", i.Message)
		}
	}
}

func TestCodeVersionInvalid(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test
@lang go
@version 1.0.0
@code_version v1.0.0
@aid_version 0.1
`)
	issues := Validate(f)
	found := false
	for _, i := range issues {
		if i.Rule == "code-version-format" {
			found = true
		}
	}
	if !found {
		t.Error("expected code-version-format warning for v1.0.0")
	}
}

func TestValidateExampleFile(t *testing.T) {
	f, _, err := parser.ParseFile("../../../../examples/http-client.aid")
	if err != nil {
		t.Skip("example file not found")
	}
	issues := Validate(f)
	for _, i := range issues {
		if i.Severity == SeverityError {
			t.Errorf("error in example file: %s", i)
		}
	}
}

func TestDecisionFieldsComplete(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test
@lang go
@version 1.0.0
@aid_version 0.1

---

@decision my_choice
@purpose Why we chose X
@chosen X
@rejected Y
@rationale X is faster
`)
	issues := Validate(f)
	for _, i := range issues {
		if i.Rule == "decision-fields" {
			t.Errorf("unexpected decision-fields issue: %s", i)
		}
	}
}

func TestDecisionFieldsMissingChosen(t *testing.T) {
	f, _, _ := parser.ParseString(`@module test
@lang go
@version 1.0.0
@aid_version 0.1

---

@decision my_choice
@purpose Why we chose X
@rationale X is faster
`)
	issues := Validate(f)
	found := false
	for _, i := range issues {
		if i.Rule == "decision-fields" && i.Entry == "decision:my_choice" {
			found = true
		}
	}
	if !found {
		t.Error("expected decision-fields warning for missing @chosen")
	}
}

func TestManifestFieldsComplete(t *testing.T) {
	f, _, _ := parser.ParseString(`@manifest
@project Test
@aid_version 0.1

---

@package query/planner
@aid_file planner.aid
@purpose Query planning
`)
	issues := Validate(f)
	for _, i := range issues {
		if i.Rule == "manifest-fields" && i.Severity == SeverityError {
			t.Errorf("unexpected manifest-fields error: %s", i)
		}
	}
}

func TestManifestFieldsMissingAidFile(t *testing.T) {
	f, _, _ := parser.ParseString(`@manifest
@project Test
@aid_version 0.1

---

@package query/planner
@purpose Query planning
`)
	issues := Validate(f)
	found := false
	for _, i := range issues {
		if i.Rule == "manifest-fields" && i.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Error("expected manifest-fields error for missing @aid_file")
	}
}
