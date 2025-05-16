---
title: "Overview"
date: 2021-03-17T10:15:16+01:00
---
# Overview

The Scope Service is a part of the [Veidemann harvester](https://github.com/NationalLibraryOfNorway/veidemann). Veidemann uses 
[gRPC](https://grpc.io/) for communication between services. The [veidemann-api](https://github.com/nlnwa/veidemann-api/tree/master/protobuf)
repository contains protobuf definitions for all services in Veidemann.

## Usage

Stubs are generated for Java and Go.
#### Java
Usage from Java:
{{< tabs groupId="java_dependency" >}}
{{% tab name="maven" %}}
1. Add repository to pom.xml
```xml
<repositories>
    <repository>
        <id>jitpack.io</id>
        <url>https://jitpack.io</url>
    </repository>
</repositories>
```
2.  Add the dependency
```xml
<dependency>
    <groupId>com.github.nlnwa</groupId>
    <artifactId>veidemann-api</artifactId>
    <version>Tag</version>
</dependency>
```
{{% /tab %}}
{{% tab name="gradle" %}}
1. Add repository in your root build.gradle at the end of repositories:
```groovy
allprojects {
  repositories {
    ...
    maven { url 'https://jitpack.io' }
  }
}
```
2. Add the dependency
```groovy
dependencies {
    implementation 'com.github.nlnwa:veidemann-api:Tag'
}
```
{{% /tab %}}
{{% tab name="sbt" %}}
Add repository in your build.sbt at the end of resolvers:
```scala
resolvers += "jitpack" at "https://jitpack.io"\
```
2. Add the dependency
```scala
libraryDependencies += "com.github.nlnwa" % "veidemann-api" % "Tag"
```
{{% /tab %}}
{{% tab name="leiningen" %}}
Add repository in your project.clj at the end of repositories:
```
:repositories [["jitpack" "https://jitpack.io"]]
```
2. Add the dependency
```
:dependencies [[com.github.nlnwa/veidemann-api "Tag"]]
```
{{% /tab %}}
{{< /tabs >}}

#### Go
To use the service from Go:
```
go get github.com/NationalLibraryOfNorway/veidemann/api
```

## Services
The Scope Service implements two of Veidemann's interfaces.
* [Scope checker service](scopechecker)
* [URI canonicalization service](uricanonicalizer)
