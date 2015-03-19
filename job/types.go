package job // import "github.com/CenturyLinkLabs/dray/job"

import (
	"crypto/md5"
	"fmt"
	"io"
	"strings"
)

// JobManager is the interface to used to represent all of the use cases which
// are necessary to manage the lifecyle of a job.
type JobManager interface {
	ListAll() ([]Job, error)
	GetByID(string) (*Job, error)
	Create(*Job) error
	Execute(*Job) error
	GetLog(*Job, int) (*JobLog, error)
	Delete(*Job) error
}

// JobRepository is the interface that wraps all of the persistence operations
// related to a job. The JobManager uses the JobRepository to maintain state
// about jobs that are submitted.
type JobRepository interface {
	All() ([]Job, error)
	Get(jobID string) (*Job, error)
	Create(job *Job) error
	Delete(jobID string) error
	Update(jobID, attr, value string) error
	GetJobLog(jobID string, index int) (*JobLog, error)
	AppendLogLine(jobID, logLine string) error
}

// JobStepExecutor is the interface that wraps the methods necessary to turn
// a job step into a running Docker container and then clean-up after the
// container has stopped.
type JobStepExecutor interface {
	Start(js *Job, stdIn io.Reader, stdOut, stdErr io.WriteCloser) error
	Inspect(js *Job) error
	CleanUp(js *Job) error
}

// Job describes the data necessary for Dray to process a job.
type Job struct {
	ID             string      `json:"id,omitempty"`
	Name           string      `json:"name,omitempty"`
	Steps          []JobStep   `json:"steps,omitempty"`
	Environment    Environment `json:"environment,omitempty"`
	StepsCompleted int         `json:"stepsCompleted,omitempty"`
	Status         string      `json:"status,omitempty"`
}

// CurrentStep returns the first JobStep from the list which has not yet
// completed execution. The StepsCompleted field on the Job struct is consulted
// in order to determine which step should be returned.
func (j Job) currentStep() *JobStep {
	return &j.Steps[j.StepsCompleted]
}

// CurrentStepEnvironment returns the complete environment for the current job
// step. The environment is constructed by merging the global, job-wide
// environment settings with the environment settings for the current step.
func (j Job) currentStepEnvironment() Environment {
	return append(j.Environment, j.currentStep().Environment...)
}

// JobStep represents one of the individual steps in a Dray Job. A job step is
// the name of the Docker image that should be executed along with some
// metadata used to control the execution of that image.
type JobStep struct {
	Name           string      `json:"name,omitempty"`
	Source         string      `json:"source,omitempty"`
	Environment    Environment `json:"environment,omitempty"`
	Output         string      `json:"output,omitempty"`
	BeginDelimiter string      `json:"beginDelimiter,omitempty"`
	EndDelimiter   string      `json:"endDelimiter,omitempty"`
	Refresh        bool        `json:"refresh,omitempty"`

	id string
}

func (js JobStep) usesStdOutPipe() bool {
	return js.Output == "stdout" || js.Output == ""
}

func (js JobStep) usesStdErrPipe() bool {
	return js.Output == "stderr"
}

func (js JobStep) usesFilePipe() bool {
	return strings.HasPrefix(js.Output, "/")
}

func (js JobStep) filePipePath() string {
	return fmt.Sprintf("/tmp/%x", md5.Sum([]byte(js.Source)))
}

func (js JobStep) usesDelimitedOutput() bool {
	return len(js.BeginDelimiter) > 0 && len(js.EndDelimiter) > 0
}

// JobLog represents the log output of a job. The Index field contains the
// starting index for the list of log lines returned while the Lines field is
// a list of log lines generated by the job.
type JobLog struct {
	Index int      `json:"index,omitempty"`
	Lines []string `json:"lines"`
}

// Environment is an array of EnvVar structs and represents the set of
// environment variables to be injected into a Docker container.
type Environment []EnvVar

// Returns an array of strings representing the environment variables in the
// list.
func (e Environment) stringify() []string {
	envStrings := make([]string, len(e))

	for i, v := range e {
		envStrings[i] = v.String()
	}

	return envStrings
}

// EnvVar represents an environment variable and its associated value.
type EnvVar struct {
	Variable string `json:"variable"`
	Value    string `json:"value"`
}

func (e EnvVar) String() string {
	return fmt.Sprintf("%s=%s", e.Variable, e.Value)
}
