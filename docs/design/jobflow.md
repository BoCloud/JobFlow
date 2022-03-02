## JobFlow

### Introduction

JobFlow defines the running flow of a set of jobs. Fields in JobFlow define how jobs are orchestrated.

### Definition

JobFlow mainly has the following fields:

* spec.jobretainpolicy: After JobFlow runs, keep the generated job. Otherwise, delete it.
* flows.name: Indicates the name of the vcjob.
* flows.dependsOn.targets: Indicates the name of the vcjob that the above vcjob depends on, which can be one or multiple vcjobs

[the sample file of JobFlow](../../example/JobFlow.yaml)