package cel_test

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/testdata/school"
	"github.com/google/cel-go/common/types/pb"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/matryer/is"
)

func makeStudentData() map[string]interface{} {
	return map[string]interface{}{
		"student.ID":             "12312",
		"student.Age":            16,
		"student.GPA":            2.2,
		"student.Status":         "Enrolled",
		"student.Grades":         []interface{}{"A", "B", "A"},
		"student.EnrollmentDate": "2018-08-03T16:00:00-07:00",
		"student.Adjustment":     2.1,
		"now":                    "2019-08-03T16:00:00-07:00",
		"specificTime":           &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}

}

func makeEducationSchema() indigo.Schema {
	return indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student.ID", Type: indigo.String{}},
			{Name: "student.Age", Type: indigo.Int{}},
			{Name: "student.GPA", Type: indigo.Float{}},
			{Name: "student.Adjustment", Type: indigo.Float{}},
			{Name: "student.Status", Type: indigo.String{}},
			{Name: "student.Grades", Type: indigo.List{ValueType: indigo.String{}}},
			{Name: "student.EnrollmentDate", Type: indigo.String{}},
			{Name: "now", Type: indigo.String{}},
			{Name: "alsoNow", Type: indigo.Timestamp{}},
		},
	}
}

func makeEducationRules() []*indigo.Rule {
	rule1 := &indigo.Rule{
		ID:     "student_actions",
		Meta:   "d04ab6d9-f59d-9474-5c38-34d65380c612",
		Schema: makeEducationSchema(),
		Rules: map[string]*indigo.Rule{
			"honors_student": {
				ID:         "honors_student",
				Expr:       `student.GPA >= 3.6 && student.Status!="Probation" && !("C" in student.Grades)`,
				ResultType: indigo.Bool{},
			},
			"at_risk": {
				ID:   "at_risk",
				Expr: `student.GPA < 2.5 || student.Status == "Probation"`,
				Rules: map[string]*indigo.Rule{
					"risk_factor": {
						ID:   "risk_factor",
						Expr: `2.0+6.0`,
					},
				},
			},
		},
	}

	rule2 := &indigo.Rule{
		ID:     "depthRules",
		Schema: makeEducationSchema(),
		Expr:   `student.GPA > 3.5`, // false
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:   "c1",
				Expr: `student.Adjustment > 0.0`, // true
			},
			"b": {
				ID:   "c2",
				Expr: `student.Adjustment > 3.0`, // false
			},
			"c": {
				ID:   "c3",
				Expr: `student.Adjustment < 2.6`, // true
			},
			"d": {
				ID:   "c4",
				Expr: `student.Adjustment > 3.0`, // false
			},
		},
	}

	rule3 := &indigo.Rule{
		ID:     "ruleOptions",
		Schema: makeEducationSchema(),
		Expr:   `student.GPA > 3.5`, // false
		Rules: map[string]*indigo.Rule{
			"A": {
				ID:       "D",
				Expr:     `student.Adjustment > 0.0`,                               // true
				EvalOpts: []indigo.EvalOption{indigo.StopFirstPositiveChild(true)}, // RULE OPTION
				Rules: map[string]*indigo.Rule{
					"d1": {
						ID:   "d1",
						Expr: `student.Adjustment < 2.6`, // true
					},
					"d2": {
						ID:   "d2",
						Expr: `student.Adjustment > 3.0`, // false
					},
					"d3": {
						ID:   "d3",
						Expr: `student.Adjustment < 2.6`, // true
					},
				},
			},
			"B": {
				ID:   "b1",
				Expr: `student.Adjustment > 3.0`, // false
			},
			"E": {
				ID:       "E",
				Expr:     `student.Adjustment > 0.0`, // true
				EvalOpts: []indigo.EvalOption{},      // NO RULE OPTION
				Rules: map[string]*indigo.Rule{
					"e1": {
						ID:   "e1",
						Expr: `student.Adjustment < 2.6`, // true
					},
					"e2": {
						ID:   "e2",
						Expr: `student.Adjustment > 3.0`, // false
					},
					"e3": {
						ID:   "e3",
						Expr: `student.Adjustment < 2.6`, // true
					},
				},
			},
		},
	}

	return []*indigo.Rule{rule1, rule2, rule3}

}

func makeEducationRulesWithIncorrectTypes() *indigo.Rule {
	rule1 := &indigo.Rule{
		ID:     "student_actions",
		Meta:   "d04ab6d9-f59d-9474-5c38-34d65380c612",
		Schema: makeEducationSchema(),
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:         "honors_student",
				Expr:       `student.GPA != "3.6" && student.Status > 2.0`,
				ResultType: indigo.Bool{},
			},
		},
	}
	return rule1
}

func TestBasicRules(t *testing.T) {

	is := is.New(t)

	evaluator := cel.NewEvaluator()
	engine := indigo.NewEngine(evaluator)
	rule := makeEducationRules()

	err := engine.AddRule(rule...)
	is.NoErr(err)

	results, err := engine.Evaluate(makeStudentData(), "student_actions")
	is.NoErr(err)
	is.Equal(results.Meta, rule[0].Meta)
	is.True(results.Pass)
	is.True(!results.Results["honors_student"].Pass)
	is.True(results.Results["at_risk"].Pass)
	is.Equal(results.Results["at_risk"].Results["risk_factor"].Value.(float64), 8.0)
}

func makeEducationProtoSchema() indigo.Schema {
	return indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "self", Type: indigo.Proto{Protoname: "school.HonorsConfiguration", Message: &school.HonorsConfiguration{}}},
		},
	}
}

func makeEducationProtoRules(id string) *indigo.Rule {
	return &indigo.Rule{
		ID:     id,
		Schema: makeEducationProtoSchema(),
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:   "honor_student",
				Expr: `student.GPA >= self.Minimum_GPA && student.Status != school.Student.status_type.PROBATION && student.Grades.all(g, g>=3.0)`,
				Self: &school.HonorsConfiguration{Minimum_GPA: 3.7},
				Meta: true,
			},
			"b": {
				ID:   "at_risk",
				Expr: `student.GPA < 2.5 || student.Status == school.Student.status_type.PROBATION`,
				Meta: false,
			},
			"c": {
				ID:   "tenure_gt_6months",
				Expr: `now - student.EnrollmentDate > duration("4320h")`, // 6 months = 4320 hours
				Meta: true,
			},
		},
	}

}

func makeStudentProtoData() map[string]interface{} {
	s := school.Student{
		Age:            16,
		GPA:            3.76,
		Status:         school.Student_ENROLLED,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	s.ProtoReflect()

	return map[string]interface{}{
		"student": &s,
		"now":     &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}

}

// Make sure that type mismatches between schema and rule are caught at compile time
func TestCompileErrors(t *testing.T) {

	is := is.New(t)

	evaluator := cel.NewEvaluator()
	engine := indigo.NewEngine(evaluator)
	rule := makeEducationRulesWithIncorrectTypes()

	err := engine.AddRule(rule)
	if err == nil {
		is.Fail() // expected compile error here
	}
	is.True(strings.Contains(err.Error(), "1:13: found no matching overload for '_!=_' applied to '(double, string)'"))
	is.True(strings.Contains(err.Error(), "1:40: found no matching overload for '_>_' applied to '(string, double)'"))
}

func TestProtoMessage(t *testing.T) {

	is := is.New(t)
	eval := cel.NewEvaluator()
	engine := indigo.NewEngine(eval, indigo.CollectDiagnostics(true), indigo.ForceDiagnosticsAllRules(true))

	err := engine.AddRule(makeEducationProtoRules("student_actions"))
	is.NoErr(err)

	results, err := engine.Evaluate(makeStudentProtoData(), "student_actions")
	is.NoErr(err)
	is.Equal(len(results.Results), 3)
	for _, v := range results.Results {
		is.Equal(v.Meta, v.Pass)
	}
}

func TestDiagnosticOptions(t *testing.T) {

	is := is.New(t)

	// Turn off diagnostic collection
	engine := indigo.NewEngine(cel.NewEvaluator(), indigo.CollectDiagnostics(false))
	err := engine.AddRule(makeEducationProtoRules("student_actions"))
	is.NoErr(err)

	_, err = engine.Evaluate(makeStudentProtoData(), "student_actions", indigo.ReturnDiagnostics(true))
	if err == nil {
		t.Errorf("Wanted error; should require indigo.CollectDiagnostics to be turned on to enable indigo.ReturnDiagnostics")
	}

	// Turn on diagnostic collection
	engine = indigo.NewEngine(cel.NewEvaluator(), indigo.CollectDiagnostics(true))
	err = engine.AddRule(makeEducationProtoRules("student_actions"))
	is.NoErr(err)

	results, err := engine.Evaluate(makeStudentProtoData(), "student_actions", indigo.ReturnDiagnostics(true))
	is.NoErr(err)
	is.Equal(results.RulesEvaluated, 4)

	for _, c := range results.Results {
		is.Equal(c.RulesEvaluated, 1)
		if len(c.Diagnostics) < 100 {
			t.Errorf("Wanted diagnostics for rule %s, got %s", c.RuleID, c.Diagnostics)
		}
	}

}

func TestRuleResultTypes(t *testing.T) {

	cases := []struct {
		rule indigo.Rule
		err  error
	}{
		{
			indigo.Rule{
				ID:         "shouldBeBool",
				Schema:     makeEducationSchema(),
				ResultType: indigo.Bool{},
				Expr:       `student.GPA >= 3.6 && student.Status!="Probation" && !("C" in student.Grades)`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "shouldBeBFloat",
				Schema:     makeEducationSchema(),
				ResultType: indigo.Float{},
				Expr:       `student.GPA + 1.0`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "shouldBeStudent",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Proto{Protoname: "school.Student"},
				Expr:       `school.Student { GPA: 1.2 }`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "NEGATIVEshouldBeBFloat",
				Schema:     makeEducationSchema(),
				ResultType: indigo.Bool{},
				Expr:       `student.GPA + 1.0`,
			},
			fmt.Errorf("Should be an error"),
		},
		{
			indigo.Rule{
				ID:         "NEGATIVEshouldBeStudent",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Proto{Protoname: "school.HonorsConfiguration"},
				Expr:       `school.Student { GPA: 1.2 }`,
			},
			fmt.Errorf("Should be an error"),
		},
	}

	eval := cel.NewEvaluator()
	engine := indigo.NewEngine(eval)

	for _, c := range cases {
		err := engine.AddRule(&c.rule)
		if c.err == nil && err != nil {
			t.Errorf("For rule %s, wanted err = %v, got %v", c.rule.ID, c.err, err)
		}
	}
}

func TestConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	is := is.New(t)
	rand.Seed(time.Now().Unix())

	e := indigo.NewEngine(cel.NewEvaluator())

	var wg sync.WaitGroup

	for i := 1; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := e.AddRule(makeEducationProtoRules(fmt.Sprintf("rule%d", i)))
			is.NoErr(err)
			r, err := e.Evaluate(makeStudentProtoData(), fmt.Sprintf("rule%d", i), indigo.ReturnDiagnostics(false))
			is.NoErr(err)
			is.Equal(r.RulesEvaluated, 4)
		}(i)
		time.Sleep(time.Duration(rand.Intn(3) * int(time.Millisecond)))
	}

	wg.Wait()
}

// // ------------------------------------------------------------------------------------------
// // BENCHMARKS
// //
// //
// //
// //
// //

func BenchmarkSimpleRuleCEL(b *testing.B) {

	engine := indigo.NewEngine(cel.NewEvaluator())

	education := makeEducationSchema()
	data := makeStudentData()

	rule := indigo.Rule{
		ID:     "student_actions",
		Schema: education,
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:     "at_risk",
				Schema: education,
				Expr:   `student.GPA < 2.5 || student.Status == "Probation"`,
			},
		},
	}

	err := engine.AddRule(&rule)
	if err != nil {
		b.Errorf("Error adding ruleset: %v", err)
	}

	for i := 0; i < b.N; i++ {
		engine.Evaluate(data, "student_actions")
	}
}

func BenchmarkSimpleRuleWithDiagnosticsCEL(b *testing.B) {

	engine := indigo.NewEngine(cel.NewEvaluator(), indigo.CollectDiagnostics(true), indigo.ForceDiagnosticsAllRules(true))
	education := makeEducationSchema()
	data := makeStudentData()

	rule := indigo.Rule{
		ID:     "student_actions",
		Schema: education,
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:     "at_risk",
				Schema: education,
				Expr:   `student.GPA < 2.5 || student.Status == "Probation"`,
			},
		},
	}

	err := engine.AddRule(&rule)
	if err != nil {
		b.Errorf("Error adding ruleset: %v", err)
	}

	for i := 0; i < b.N; i++ {
		engine.Evaluate(data, "student_actions")
	}
}

func BenchmarkRuleWithArrayCEL(b *testing.B) {

	engine := indigo.NewEngine(cel.NewEvaluator())
	education := makeEducationSchema()

	rule := indigo.Rule{
		ID:     "student_actions",
		Schema: education,
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:     "honors_student",
				Schema: education,
				Expr:   `student.GPA >= 3.6 && student.Status!="Probation" && !("C" in student.Grades)`,
			},
		},
	}

	err := engine.AddRule(&rule)
	if err != nil {
		b.Errorf("Error adding ruleset: %v", err)
	}

	data := makeStudentData()
	for i := 0; i < b.N; i++ {
		engine.Evaluate(data, "student_actions")
	}
}

func BenchmarkProtoWithSelfXCEL(b *testing.B) {
	b.StopTimer()

	pb.DefaultDb.RegisterMessage(&school.Student{})

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "self", Type: indigo.Proto{Protoname: "school.HonorsConfiguration", Message: &school.HonorsConfiguration{}}},
		},
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	rule := indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:   "at_risk",
				Expr: `student.GPA < self.Minimum_GPA && student.Status == school.Student.status_type.PROBATION`,
				Self: &school.HonorsConfiguration{Minimum_GPA: 3.7},
				Meta: false,
			},
		},
	}

	err := engine.AddRule(&rule)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		GPA:            3,
		Status:         school.Student_PROBATION,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]interface{}{
		"student": &s,
		"now":     &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		engine.Evaluate(data, "student_actions")
	}

}

func BenchmarkProtoWithoutSelfCEL(b *testing.B) {

	pb.DefaultDb.RegisterMessage(&school.Student{})

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
		},
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	rule := indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:   "at_risk",
				Expr: `student.GPA < 2.5 || student.Status == school.Student.status_type.PROBATION`,
				Meta: false,
			},
		},
	}

	err := engine.AddRule(&rule)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		GPA:            3.76,
		Status:         school.Student_ENROLLED,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]interface{}{
		"student": &s,
		"now":     &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}

	for i := 0; i < b.N; i++ {
		engine.Evaluate(data, "student_actions")
	}

}

func BenchmarkProtoCreationCEL(b *testing.B) {
	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "student_suspension", Type: indigo.Proto{Protoname: "school.Student.Suspension", Message: &school.Student_Suspension{}}},
			{Name: "studentSummary", Type: indigo.Proto{Protoname: "school.StudentSummary", Message: &school.StudentSummary{}}},
		},
	}

	// data := map[string]interface{}{
	// 	"student": school.Student{
	// 		Grades: []float64{3.0, 2.9, 4.0, 2.1},
	// 		Suspensions: []*school.Student_Suspension{
	// 			&school.Student_Suspension{Cause: "Cheating"},
	// 			&school.Student_Suspension{Cause: "Fighting"},
	// 		},
	// 	},
	// }

	rule := indigo.Rule{
		ID:         "create_summary",
		Schema:     education,
		ResultType: indigo.Proto{Protoname: "school.StudentSummary", Message: &school.StudentSummary{}},
		Expr: `
			school.StudentSummary {
				GPA: student.GPA,
				RiskFactor: 2.0 + 3.0,
				Tenure: duration("12h")
			}`,
	}

	evaluator := cel.NewEvaluator()
	engine := indigo.NewEngine(evaluator)
	err := engine.AddRule(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	for i := 0; i < b.N; i++ {
		engine.Evaluate(map[string]interface{}{}, "create_summary")
	}

}

func BenchmarkProto20KXCEL(b *testing.B) {
	b.StopTimer()
	pb.DefaultDb.RegisterMessage(&school.Student{})

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "self", Type: indigo.Proto{Protoname: "school.HonorsConfiguration", Message: &school.HonorsConfiguration{}}},
		},
	}

	engine := indigo.NewEngine(cel.NewEvaluator(), indigo.CollectDiagnostics(false))

	rule := &indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules:  map[string]*indigo.Rule{},
	}

	for i := 0; i < 2_000; i++ {
		r := &indigo.Rule{
			ID:   fmt.Sprintf("at_risk_%d", i),
			Expr: `student.GPA < self.Minimum_GPA && student.Status == school.Student.status_type.PROBATION`,
			Self: &school.HonorsConfiguration{Minimum_GPA: 3.7},
			Meta: false,
		}
		rule.AddChild(r)
	}

	err := engine.AddRule(rule)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		GPA:            3,
		Status:         school.Student_PROBATION,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]interface{}{
		"student": &s,
		"now":     &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		engine.Evaluate(data, "student_actions")
	}
}
