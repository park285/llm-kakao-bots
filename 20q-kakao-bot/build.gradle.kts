import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import java.util.concurrent.TimeUnit

plugins {
    id("org.springframework.boot") version "4.0.0"
    id("io.spring.dependency-management") version "1.1.7"
    kotlin("jvm") version "2.3.0-RC3"
    kotlin("plugin.spring") version "2.3.0-RC3"
    id("org.jlleitschuh.gradle.ktlint") version "13.1.0"
}

group = "party.qwer"
version = "0.0.1-SNAPSHOT"

java {
    toolchain {
        languageVersion.set(JavaLanguageVersion.of(25))
    }
}

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

dependencyManagement {
    imports {
        mavenBom("org.springframework.boot:spring-boot-dependencies:4.0.0")
        mavenBom("org.jetbrains.kotlinx:kotlinx-coroutines-bom:1.10.2")
        mavenBom("io.ktor:ktor-bom:3.2.0")
    }
}

dependencies {
    implementation("org.springframework.boot:spring-boot-starter-webflux")
    implementation("org.springframework.boot:spring-boot-starter-data-redis")
    implementation("org.redisson:redisson-spring-boot-starter:3.52.0")
    implementation("org.springframework.boot:spring-boot-starter-actuator")
    implementation("org.springframework.boot:spring-boot-starter-cache")
    
    // R2DBC PostgreSQL - 사용자 스탯 저장
    implementation("org.springframework.boot:spring-boot-starter-data-r2dbc")
    implementation("org.postgresql:r2dbc-postgresql:1.0.7.RELEASE")
    implementation("org.jetbrains.kotlin:kotlin-reflect")
    implementation("me.paulschwarz:spring-dotenv:4.0.0")
    implementation("tools.jackson.module:jackson-module-kotlin")
    implementation("tools.jackson.dataformat:jackson-dataformat-yaml")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-reactor")
    // Ktor Client - REST API 통신 (OkHttp: h2c)
    implementation("io.ktor:ktor-client-okhttp")
    implementation("io.ktor:ktor-client-cio")
    implementation("io.ktor:ktor-client-content-negotiation")
    implementation("io.ktor:ktor-serialization-jackson")
    implementation("com.google.re2j:re2j:1.7")
    implementation("org.ahocorasick:ahocorasick:0.6.3")
    implementation("com.ibm.icu:icu4j:78.1")
    implementation("com.github.ben-manes.caffeine:caffeine:3.2.3")
    implementation("com.google.guava:guava:33.5.0-jre")
    implementation("com.vladsch.flexmark:flexmark:0.64.8")
    implementation("com.sigpwned:emoji4j:16.0.0")
    implementation("com.squareup.okhttp3:okhttp:5.3.2")
    implementation("com.networknt:json-schema-validator:2.0.0")
    
    // TOON Format - LLM 프롬프트 최적화 (30-60% 토큰 절감)
    implementation("dev.toonformat:jtoon:1.0.5")
    
    testImplementation("org.springframework.boot:spring-boot-starter-test")
    testImplementation("io.mockk:mockk:1.14.6")
    testImplementation("com.ninja-squad:springmockk:4.0.2")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test")
    testImplementation("io.projectreactor:reactor-test")
    testImplementation("io.ktor:ktor-client-mock")
    testImplementation("net.bytebuddy:byte-buddy:1.18.2")
    testImplementation("net.bytebuddy:byte-buddy-agent:1.18.2")
}

kotlin {
    compilerOptions {
        jvmTarget.set(JvmTarget.JVM_25)
        
        // Progressive mode: 미래 언어 기능 활성화, 경고를 에러로 처리
        progressiveMode.set(true)
        
        // 컴파일러 인수
        freeCompilerArgs.addAll(
            // Null 허용성
            "-Xjsr305=strict",
            
            // Kotlin 2.3.0 신규 기능
            "-Xreturn-value-checker=check",           // 미사용 반환값 경고
            "-Xsuppress-version-warnings",            // Beta 버전 경고 억제
            
            
            // 실험적 기능
            "-opt-in=kotlin.RequiresOptIn",           // Opt-in API 자동 활성화
            "-opt-in=kotlin.ExperimentalStdlibApi",   // 실험적 stdlib API
            "-opt-in=kotlinx.coroutines.ExperimentalCoroutinesApi",  // 실험적 coroutines API
            "-Xcontext-parameters",                   // Context 파라미터 (Kotlin 2.3.0)
        )
    }
}

tasks.withType<JavaCompile> {
    options.compilerArgs.addAll(listOf(
        "--add-modules=jdk.incubator.vector"
    ))
}

tasks.withType<Test> {
    useJUnitPlatform()
    
    // 통합 테스트 제외 (일반 빌드에서는 유닛 테스트만 실행)
    exclude("**/*IntegrationTest.class")
    
    jvmArgs(
        "-Xshare:off",
        "--enable-native-access=ALL-UNNAMED"
    )
    systemProperty("spring.profiles.active", "test")
}

// 통합 테스트 전용 태스크
val integrationTest by tasks.registering(Test::class) {
    description = "Run integration tests"
    group = "verification"
    
    testClassesDirs = sourceSets["test"].output.classesDirs
    classpath = sourceSets["test"].runtimeClasspath
    
    useJUnitPlatform()
    shouldRunAfter(tasks.test)
    
    // 통합 테스트만 실행
    include("**/*IntegrationTest.class")
    
    jvmArgs(
        "-Xshare:off",
        "--enable-native-access=ALL-UNNAMED"
    )
    systemProperty("spring.profiles.active", "integration")
}

tasks.named<org.springframework.boot.gradle.tasks.run.BootRun>("bootRun") {
    jvmArgs(
        "--enable-native-access=ALL-UNNAMED"
    )
}

tasks.named<org.springframework.boot.gradle.tasks.bundling.BootJar>("bootJar") {
    requiresUnpack("**/kotlin-reflect-*.jar")
    requiresUnpack("**/kotlin-stdlib-*.jar")
}

ktlint {
    version.set("1.7.1")
    android.set(false)
    outputColorName.set("RED")

    this.enableExperimentalRules.set(false)

    filter {
        exclude("**/src/test/**")
        exclude("**/*.gradle.kts")
        exclude("**/build/**")
    }
}

tasks.withType<org.jlleitschuh.gradle.ktlint.tasks.BaseKtLintCheckTask>().configureEach {
    exclude { it.file.absolutePath.contains("/build/") }
}

val detektVersion = "2.0.0-alpha.1"
val detektCliJar = "detekt-cli-${detektVersion}-all.jar"
val detektDir = file(".detekt")
val detektJarFile = file("${detektDir}/${detektCliJar}")

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
        val downloadUrl = "https://github.com/detekt/detekt/releases/download/v${version}/${jar}"
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

// Detekt CLI 실행 task
val detektCli by tasks.registering(Exec::class) {
    description = "Run Detekt static analysis using CLI JAR"
    group = "verification"

    dependsOn(downloadDetektCli)

    val reportDir = layout.buildDirectory.dir("reports/detekt")
    outputs.dir(reportDir)

    // Configuration Cache 호환: 외부 변수 참조 제거, 직접 정의
    val version = "2.0.0-alpha.1"
    val jarPath = file(".detekt/detekt-cli-${version}-all.jar").absolutePath

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
        "--language-version", "2.3"
    )

    doLast {
        val reportFile = reportDir.get().file("detekt.html").asFile
        if (reportFile.exists()) {
            println("Detekt 분석 완료! Report: ${reportFile.absolutePath}")
        }
    }
}

// check task에 detektCli 의존성 추가 (비활성화: Configuration Cache 충돌)
// CI/CD에서는 scripts/detekt-cli.sh를 직접 실행
// tasks.named("check") {
//     dependsOn(detektCli)
// }
