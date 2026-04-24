package taskfields

import (
	"strings"
)

const (
	TaskContext                = "task_context"
	WhatToDo                   = "what_to_do"
	ConstraintsAndDependencies = "constraints_and_dependencies"
	ReadinessCriteria          = "readiness_criteria"
	WhatIsBroken               = "what_is_broken"
	ReproductionSteps          = "reproduction_steps"
	ExpectedBehavior           = "expected_behavior"
	ActualBehavior             = "actual_behavior"
	Environment                = "environment"
	ImpactAndRisks             = "impact_and_risks"
	SuspectedCause             = "suspected_cause"
	FixScope                   = "fix_scope"
	VerificationCriteria       = "verification_criteria"
	DesignOrDocsLinks          = "design_or_docs_links"
	Prerequisites              = "prerequisites"
	ProblemToSolve             = "problem_to_solve"
	BriefSolution              = "brief_solution"
	Risks                      = "risks"
	Approvers                  = "approvers"
	ProjectParticipants        = "project_participants"
	AcceptanceCriteria         = "acceptance_criteria"
	UsefulLinks                = "useful_links"
	DueDate                    = "due_date"
	SelectedLinks              = "selected_links"
)

type TaskFields struct {
	TaskContext                string `json:"task_context,omitempty"`
	WhatToDo                   string `json:"what_to_do,omitempty"`
	ConstraintsAndDependencies string `json:"constraints_and_dependencies,omitempty"`
	ReadinessCriteria          string `json:"readiness_criteria,omitempty"`
	WhatIsBroken               string `json:"what_is_broken,omitempty"`
	ReproductionSteps          string `json:"reproduction_steps,omitempty"`
	ExpectedBehavior           string `json:"expected_behavior,omitempty"`
	ActualBehavior             string `json:"actual_behavior,omitempty"`
	Environment                string `json:"environment,omitempty"`
	ImpactAndRisks             string `json:"impact_and_risks,omitempty"`
	SuspectedCause             string `json:"suspected_cause,omitempty"`
	FixScope                   string `json:"fix_scope,omitempty"`
	VerificationCriteria       string `json:"verification_criteria,omitempty"`
	DesignOrDocsLinks          string `json:"design_or_docs_links,omitempty"`
	Prerequisites              string `json:"prerequisites,omitempty"`
	ProblemToSolve             string `json:"problem_to_solve,omitempty"`
	BriefSolution              string `json:"brief_solution,omitempty"`
	Risks                      string `json:"risks,omitempty"`
	Approvers                  string `json:"approvers,omitempty"`
	ProjectParticipants        string `json:"project_participants,omitempty"`
	AcceptanceCriteria         string `json:"acceptance_criteria,omitempty"`
	UsefulLinks                string `json:"useful_links,omitempty"`
}

type FieldDefinition struct {
	Key         string `yaml:"key"`
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
}

func KnownDefinitions() []FieldDefinition {
	return []FieldDefinition{
		{Key: TaskContext, Label: "Контекст задачи"},
		{Key: WhatToDo, Label: "Что нужно сделать"},
		{Key: ConstraintsAndDependencies, Label: "Ограничения и зависимости"},
		{Key: ReadinessCriteria, Label: "Критерии готовности"},
		{Key: WhatIsBroken, Label: "Что сломано"},
		{Key: ReproductionSteps, Label: "Шаги воспроизведения"},
		{Key: ExpectedBehavior, Label: "Ожидаемое поведение"},
		{Key: ActualBehavior, Label: "Фактическое поведение"},
		{Key: Environment, Label: "Окружение"},
		{Key: ImpactAndRisks, Label: "Влияние и риски"},
		{Key: SuspectedCause, Label: "Предполагаемая причина"},
		{Key: FixScope, Label: "Что нужно исправить"},
		{Key: VerificationCriteria, Label: "Критерии проверки"},
		{Key: DesignOrDocsLinks, Label: "Ссылки на макет или документацию"},
		{Key: Prerequisites, Label: "Предпосылки задачи"},
		{Key: ProblemToSolve, Label: "Проблема, которую решаем"},
		{Key: BriefSolution, Label: "Краткое описание решения"},
		{Key: Risks, Label: "Риски"},
		{Key: Approvers, Label: "Согласующие"},
		{Key: ProjectParticipants, Label: "Участники проекта"},
		{Key: AcceptanceCriteria, Label: "Критерии приемки"},
		{Key: UsefulLinks, Label: "Полезные ссылки"},
	}
}

func KnownKeys() []string {
	defs := KnownDefinitions()
	keys := make([]string, 0, len(defs))
	for _, def := range defs {
		keys = append(keys, def.Key)
	}
	return keys
}

func IsKnownKey(key string) bool {
	_, ok := definitionByKey()[strings.TrimSpace(key)]
	return ok
}

func IsKnownTemplateKey(key string) bool {
	key = strings.TrimSpace(key)
	return IsKnownKey(key) || key == DueDate || key == SelectedLinks
}

func LabelForKey(key string) string {
	switch strings.TrimSpace(key) {
	case DueDate:
		return "Срок"
	case SelectedLinks:
		return "Полезные материалы"
	}
	if def, ok := definitionByKey()[strings.TrimSpace(key)]; ok {
		return def.Label
	}
	return ""
}

func (f TaskFields) Value(key string) string {
	switch strings.TrimSpace(key) {
	case TaskContext:
		return strings.TrimSpace(f.TaskContext)
	case WhatToDo:
		return strings.TrimSpace(f.WhatToDo)
	case ConstraintsAndDependencies:
		return strings.TrimSpace(f.ConstraintsAndDependencies)
	case ReadinessCriteria:
		return strings.TrimSpace(f.ReadinessCriteria)
	case WhatIsBroken:
		return strings.TrimSpace(f.WhatIsBroken)
	case ReproductionSteps:
		return strings.TrimSpace(f.ReproductionSteps)
	case ExpectedBehavior:
		return strings.TrimSpace(f.ExpectedBehavior)
	case ActualBehavior:
		return strings.TrimSpace(f.ActualBehavior)
	case Environment:
		return strings.TrimSpace(f.Environment)
	case ImpactAndRisks:
		return strings.TrimSpace(f.ImpactAndRisks)
	case SuspectedCause:
		return strings.TrimSpace(f.SuspectedCause)
	case FixScope:
		return strings.TrimSpace(f.FixScope)
	case VerificationCriteria:
		return strings.TrimSpace(f.VerificationCriteria)
	case DesignOrDocsLinks:
		return strings.TrimSpace(f.DesignOrDocsLinks)
	case Prerequisites:
		return strings.TrimSpace(f.Prerequisites)
	case ProblemToSolve:
		return strings.TrimSpace(f.ProblemToSolve)
	case BriefSolution:
		return strings.TrimSpace(f.BriefSolution)
	case Risks:
		return strings.TrimSpace(f.Risks)
	case Approvers:
		return strings.TrimSpace(f.Approvers)
	case ProjectParticipants:
		return strings.TrimSpace(f.ProjectParticipants)
	case AcceptanceCriteria:
		return strings.TrimSpace(f.AcceptanceCriteria)
	case UsefulLinks:
		return strings.TrimSpace(f.UsefulLinks)
	default:
		return ""
	}
}

func (f TaskFields) FilledDefinitions() []FieldDefinition {
	result := make([]FieldDefinition, 0)
	for _, def := range KnownDefinitions() {
		if f.Value(def.Key) != "" {
			result = append(result, def)
		}
	}
	return result
}

func (f TaskFields) Clean() TaskFields {
	return TaskFields{
		TaskContext:                strings.TrimSpace(f.TaskContext),
		WhatToDo:                   strings.TrimSpace(f.WhatToDo),
		ConstraintsAndDependencies: strings.TrimSpace(f.ConstraintsAndDependencies),
		ReadinessCriteria:          strings.TrimSpace(f.ReadinessCriteria),
		WhatIsBroken:               strings.TrimSpace(f.WhatIsBroken),
		ReproductionSteps:          strings.TrimSpace(f.ReproductionSteps),
		ExpectedBehavior:           strings.TrimSpace(f.ExpectedBehavior),
		ActualBehavior:             strings.TrimSpace(f.ActualBehavior),
		Environment:                strings.TrimSpace(f.Environment),
		ImpactAndRisks:             strings.TrimSpace(f.ImpactAndRisks),
		SuspectedCause:             strings.TrimSpace(f.SuspectedCause),
		FixScope:                   strings.TrimSpace(f.FixScope),
		VerificationCriteria:       strings.TrimSpace(f.VerificationCriteria),
		DesignOrDocsLinks:          strings.TrimSpace(f.DesignOrDocsLinks),
		Prerequisites:              strings.TrimSpace(f.Prerequisites),
		ProblemToSolve:             strings.TrimSpace(f.ProblemToSolve),
		BriefSolution:              strings.TrimSpace(f.BriefSolution),
		Risks:                      strings.TrimSpace(f.Risks),
		Approvers:                  strings.TrimSpace(f.Approvers),
		ProjectParticipants:        strings.TrimSpace(f.ProjectParticipants),
		AcceptanceCriteria:         strings.TrimSpace(f.AcceptanceCriteria),
		UsefulLinks:                strings.TrimSpace(f.UsefulLinks),
	}
}

func LowerLabelForKey(key string) string {
	label := LabelForKey(key)
	if label == "" {
		return ""
	}
	return strings.ToLower(label)
}

func definitionByKey() map[string]FieldDefinition {
	defs := KnownDefinitions()
	result := make(map[string]FieldDefinition, len(defs))
	for _, def := range defs {
		result[def.Key] = def
	}
	return result
}
