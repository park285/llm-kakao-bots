import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import java.util.concurrent.TimeUnit

plugins {
    kotlin("jvm") version "2.3.0-RC3"
    kotlin("plugin.serialization") version "2.3.0-RC3"
    id("io.ktor.plugin") version "3.3.2"
    id("org.jlleitschuh.gradle.ktlint") version "13.1.0"
    application
}

group = "io.github.kapu"
version = "1.0.0"

repositories {
    mavenCentral()
    maven { url = uri("https://jitpack.io") }
}

configurations.all {
    resolutionStrategy {
        cacheDynamicVersionsFor(4, TimeUnit.HOURS)
        cacheChangingModulesFor(10, TimeUnit.MINUTES)
    }
}

val ktorVersion = "3.3.2"
val koinVersion = "4.0.0"
// MCP SDK
val mcpSdkVersion = "0.5.0"
val redissonVersion = "3.52.0"
val logbackVersion = "1.5.12"
val kotlinLoggingVersion = "7.0.0"
val kotestVersion = "5.9.1"
val mockkVersion = "1.13.13"

// Security
val ahoCorasickVersion = "0.6.3"
val re2jVersion = "1.7"
val icu4jVersion = "76.1"

dependencies {
    // Kotlin
    implementation(kotlin("stdlib"))
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.9.0")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-reactor:1.9.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.3")
    implementation("org.jetbrains.kotlinx:kotlinx-io-core:0.6.0")

    // Ktor Server (Netty)
    implementation("io.ktor:ktor-server-core:$ktorVersion")
    implementation("io.ktor:ktor-server-netty:$ktorVersion")
    implementation("io.ktor:ktor-server-content-negotiation:$ktorVersion")
    implementation("io.ktor:ktor-serialization-kotlinx-json:$ktorVersion")
    implementation("io.ktor:ktor-server-call-logging:$ktorVersion")
    implementation("io.ktor:ktor-server-status-pages:$ktorVersion")
    implementation("io.ktor:ktor-server-cors:$ktorVersion")

    // Ktor Client (for external APIs)
    implementation("io.ktor:ktor-client-core:$ktorVersion")
    implementation("io.ktor:ktor-client-cio:$ktorVersion")
    implementation("io.ktor:ktor-client-okhttp:$ktorVersion")
    implementation("io.ktor:ktor-client-content-negotiation:$ktorVersion")

    // Koin (DI)
    implementation("io.insert-koin:koin-core:$koinVersion")
    implementation("io.insert-koin:koin-ktor:$koinVersion")
    implementation("io.insert-koin:koin-logger-slf4j:$koinVersion")

    // MCP SDK (Model Context Protocol client)
    implementation("io.modelcontextprotocol:kotlin-sdk:$mcpSdkVersion")

    // Redisson (Valkey 9.0)
    implementation("org.redisson:redisson:$redissonVersion")

    // Validation (Konform)
    implementation("io.konform:konform-jvm:0.6.1")

    // Logging
    implementation("io.github.oshai:kotlin-logging-jvm:$kotlinLoggingVersion")
    implementation("ch.qos.logback:logback-classic:$logbackVersion")

    // Config
    implementation("com.typesafe:config:1.4.3")
    implementation("io.github.cdimascio:dotenv-kotlin:6.4.2")

    // YAML
    implementation("com.charleskorn.kaml:kaml:0.61.0")

    // Security (Aho-Corasick, RE2j, ICU4J)
    implementation("org.ahocorasick:ahocorasick:$ahoCorasickVersion")
    implementation("com.google.re2j:re2j:$re2jVersion")
    implementation("com.ibm.icu:icu4j:$icu4jVersion")

    // Testing
    testImplementation("io.kotest:kotest-runner-junit5:$kotestVersion")
    testImplementation("io.kotest:kotest-assertions-core:$kotestVersion")
    testImplementation("io.kotest:kotest-property:$kotestVersion")
    testImplementation("io.mockk:mockk:$mockkVersion")
    testImplementation("io.ktor:ktor-server-test-host:$ktorVersion")
    testImplementation("io.insert-koin:koin-test:$koinVersion")
}

application {
    mainClass.set("io.github.kapu.turtlesoup.ApplicationKt")
}

kotlin {
    compilerOptions {
        jvmTarget.set(JvmTarget.JVM_25)

        // Progressive mode
        progressiveMode.set(true)

        freeCompilerArgs.addAll(
            "-Xjsr305=strict",
            "-opt-in=kotlin.RequiresOptIn",
            "-Xreturn-value-checker=check",
            "-Xsuppress-version-warnings",
        )
    }
}

java {
    toolchain {
        languageVersion.set(JavaLanguageVersion.of(25))
    }
}

tasks.withType<Test> {
    useJUnitPlatform()
}

ktlint {
    version.set("1.0.1")
    android.set(false)
    outputToConsole.set(true)
    ignoreFailures.set(false)
}

// Detekt CLI JAR 다운로드 및 실행
val detektVersion = "2.0.0-alpha.1"
val detektCliJar = "detekt-cli-$detektVersion-all.jar"
val detektDir = file(".detekt")
val detektJarFile = file("$detektDir/$detektCliJar")

val downloadDetektCli by tasks.registering(Exec::class) {
    description = "Download Detekt CLI JAR"
    group = "verification"

    val jarFile = detektJarFile
    val dir = detektDir
    val version = detektVersion
    val jar = detektCliJar

    outputs.file(jarFile)
    onlyIf { !jarFile.exists() }

    doFirst {
        dir.mkdirs()
        val downloadUrl = "https://github.com/detekt/detekt/releases/download/v$version/$jar"
        println("Detekt $version 다운로드 중...")
        commandLine("curl", "-L", "-o", jarFile.absolutePath, downloadUrl)
    }

    doLast {
        if (jarFile.exists()) {
            println("Detekt CLI 다운로드 완료: ${jarFile.name}")
        } else {
            throw GradleException("Detekt CLI 다운로드 실패")
        }
    }
}

val detektCli by tasks.registering(Exec::class) {
    description = "Run Detekt static analysis using CLI JAR"
    group = "verification"

    dependsOn(downloadDetektCli)

    val reportDir = layout.buildDirectory.dir("reports/detekt")
    outputs.dir(reportDir)

    val version = "2.0.0-alpha.1"
    val jarPath = file(".detekt/detekt-cli-$version-all.jar").absolutePath

    doFirst {
        reportDir.get().asFile.mkdirs()
        println("Detekt $version 실행 중...")
    }

    commandLine(
        "java", "-jar", jarPath,
        "--config", "config/detekt/detekt.yml",
        "--input", "src/main/kotlin",
        "--report", "html:build/reports/detekt/detekt.html",
        "--report", "xml:build/reports/detekt/detekt.xml",
        "--build-upon-default-config",
        "--jvm-target", "24",
        "--language-version", "2.3",
    )

    doLast {
        val reportFile = reportDir.get().file("detekt.html").asFile
        if (reportFile.exists()) {
            println("Detekt 분석 완료! Report: ${reportFile.absolutePath}")
        }
    }
}

ktor {
    docker {
        jreVersion.set(JavaVersion.VERSION_25)
        localImageName.set("turtle-soup-bot")
        imageTag.set(version.toString())
    }
}
