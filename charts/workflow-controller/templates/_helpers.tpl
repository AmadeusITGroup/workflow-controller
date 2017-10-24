{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a docker images name 
*/}}
{{- define "docker-image" -}}
{{- $name := default .Chart.Name .Values.image.name -}}
{{- $tag := default .Chart.Version .Values.image.tag -}}
{{- printf "%s%s/%s:%s" .Values.image.registry .Values.image.account $name $tag -}}
{{- end -}}
