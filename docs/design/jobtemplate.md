## JobTemplate

### Introduction

* JobTemplate is the template of vcjob, after JobTemplate is created, it will not be processed by vc-controller like vcjob, it will wait to be referenced by JobFlow.
* JobTemplate can be referenced multiple times  by JobFlow.
* JobTemplate can be converted to and from vcjob.

### Definition

The spec field of JobTemplate is exactly the same as that of vcjob. You can view [the sample file of JobTemplate](../../example/JobTemplate.yaml)