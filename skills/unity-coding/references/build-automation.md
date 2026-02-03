# Build Automation

CI/CD 통합, CLI 빌드, iOS 포스트 프로세싱.

## Table of Contents
1. [BuildAutomation](#buildautomation) - 자동화 빌드
2. [iOSPostProcessBuild](#iospostprocessbuild) - Xcode 자동 설정
3. [CI/CD 연동](#cicd-연동)

---

## BuildAutomation

Unity Editor 메뉴 및 CLI에서 호출 가능한 빌드 자동화.

```csharp
#if UNITY_EDITOR
using UnityEditor;
using UnityEditor.Build.Reporting;
using UnityEngine;
using System;
using System.Linq;

/// <summary>
/// Automated build scripts for iOS and Android
/// Can be called from command line or menu
/// </summary>
public static class BuildAutomation
{
    private static string[] GetScenes()
    {
        return EditorBuildSettings.scenes
            .Where(s => s.enabled)
            .Select(s => s.path)
            .ToArray();
    }

    [MenuItem("Build/Build iOS (Development)")]
    public static void BuildiOSDev()
    {
        BuildiOS(true);
    }

    [MenuItem("Build/Build iOS (Release)")]
    public static void BuildiOSRelease()
    {
        BuildiOS(false);
    }

    public static void BuildiOS(bool development = false)
    {
        var options = new BuildPlayerOptions
        {
            scenes = GetScenes(),
            locationPathName = "Builds/iOS",
            target = BuildTarget.iOS,
            options = development ? BuildOptions.Development : BuildOptions.None
        };

        Build(options);
    }

    [MenuItem("Build/Build Android APK (Development)")]
    public static void BuildAndroidAPKDev()
    {
        BuildAndroid(true, false);
    }

    [MenuItem("Build/Build Android AAB (Release)")]
    public static void BuildAndroidAABRelease()
    {
        BuildAndroid(false, true);
    }

    public static void BuildAndroid(bool development = false, bool appBundle = false)
    {
        // Configure keystore from environment variables (for CI/CD)
        string keystorePass = Environment.GetEnvironmentVariable("KEYSTORE_PASS");
        string keyaliasPass = Environment.GetEnvironmentVariable("KEY_PASS");

        if (!string.IsNullOrEmpty(keystorePass))
        {
            PlayerSettings.Android.keystoreName = "ProjectSettings/keystore.keystore";
            PlayerSettings.Android.keystorePass = keystorePass;
            PlayerSettings.Android.keyaliasName = "release";
            PlayerSettings.Android.keyaliasPass = keyaliasPass ?? keystorePass;
        }

        EditorUserBuildSettings.buildAppBundle = appBundle;

        string extension = appBundle ? "aab" : "apk";
        string fileName = $"game.{extension}";

        var options = new BuildPlayerOptions
        {
            scenes = GetScenes(),
            locationPathName = $"Builds/Android/{fileName}",
            target = BuildTarget.Android,
            options = development ? BuildOptions.Development : BuildOptions.None
        };

        Build(options);
    }

    private static void Build(BuildPlayerOptions options)
    {
        Debug.Log($"Starting build: {options.target}");

        BuildReport report = BuildPipeline.BuildPlayer(options);
        BuildSummary summary = report.summary;

        if (summary.result == BuildResult.Succeeded)
        {
            Debug.Log($"Build succeeded: {summary.totalSize / (1024 * 1024):F2} MB");
            Debug.Log($"Output: {summary.outputPath}");
        }
        else
        {
            Debug.LogError($"Build failed: {summary.result}");
            foreach (var step in report.steps)
            {
                foreach (var message in step.messages)
                {
                    if (message.type == LogType.Error)
                        Debug.LogError(message.content);
                }
            }

            // Exit with error code for CI/CD
            if (Application.isBatchMode)
                EditorApplication.Exit(1);
        }
    }

    // Command line entry points
    public static void CLI_BuildiOS() => BuildiOS(false);
    public static void CLI_BuildAndroid() => BuildAndroid(false, true);
}
#endif
```

### CLI 사용법
```bash
# iOS Release 빌드
Unity -batchmode -quit -projectPath . -executeMethod BuildAutomation.CLI_BuildiOS

# Android AAB 빌드 (keystore 포함)
KEYSTORE_PASS=mypass KEY_PASS=mykey Unity -batchmode -quit -projectPath . -executeMethod BuildAutomation.CLI_BuildAndroid
```

---

## iOSPostProcessBuild

빌드 후 Xcode 프로젝트 자동 설정. ATT, 프레임워크, Info.plist.

```csharp
#if UNITY_IOS
using UnityEditor;
using UnityEditor.Callbacks;
using UnityEditor.iOS.Xcode;
using System.IO;

/// <summary>
/// Automatically configures Xcode project after Unity build
/// Add frameworks, Info.plist keys, build settings
/// </summary>
public class iOSPostProcessBuild
{
    [PostProcessBuild(100)]
    public static void OnPostProcessBuild(BuildTarget target, string pathToBuiltProject)
    {
        if (target != BuildTarget.iOS) return;

        ModifyPlist(pathToBuiltProject);
        ModifyProject(pathToBuiltProject);
    }

    static void ModifyPlist(string path)
    {
        string plistPath = Path.Combine(path, "Info.plist");
        PlistDocument plist = new PlistDocument();
        plist.ReadFromFile(plistPath);

        PlistElementDict root = plist.root;

        // App Transport Security (allow HTTP if needed)
        // root.CreateDict("NSAppTransportSecurity").SetBoolean("NSAllowsArbitraryLoads", true);

        // Export compliance (no encryption)
        root.SetBoolean("ITSAppUsesNonExemptEncryption", false);

        // Privacy descriptions (uncomment as needed)
        // root.SetString("NSCameraUsageDescription", "Camera access for AR features");
        // root.SetString("NSPhotoLibraryUsageDescription", "Save screenshots");
        // root.SetString("NSLocationWhenInUseUsageDescription", "Location for gameplay");

        // ATT (App Tracking Transparency)
        root.SetString("NSUserTrackingUsageDescription",
            "This identifier will be used to deliver personalized ads to you.");

        plist.WriteToFile(plistPath);
    }

    static void ModifyProject(string path)
    {
        string projectPath = PBXProject.GetPBXProjectPath(path);
        PBXProject project = new PBXProject();
        project.ReadFromFile(projectPath);

        string mainTarget = project.GetUnityMainTargetGuid();
        string frameworkTarget = project.GetUnityFrameworkTargetGuid();

        // Disable Bitcode (deprecated)
        project.SetBuildProperty(mainTarget, "ENABLE_BITCODE", "NO");
        project.SetBuildProperty(frameworkTarget, "ENABLE_BITCODE", "NO");

        // Add frameworks
        project.AddFrameworkToProject(mainTarget, "AdSupport.framework", false);
        project.AddFrameworkToProject(mainTarget, "AppTrackingTransparency.framework", true);
        project.AddFrameworkToProject(mainTarget, "StoreKit.framework", false);

        // Add capabilities (Push Notifications, In-App Purchase)
        // project.AddCapability(mainTarget, PBXCapabilityType.PushNotifications);
        // project.AddCapability(mainTarget, PBXCapabilityType.InAppPurchase);

        project.WriteToFile(projectPath);
    }
}
#endif
```

### 자동 설정 항목

| 항목 | 설정 | 용도 |
|------|------|------|
| ITSAppUsesNonExemptEncryption | false | 수출 규정 간소화 |
| NSUserTrackingUsageDescription | 광고 추적 설명 | ATT 팝업 |
| ENABLE_BITCODE | NO | 빌드 속도, 호환성 |
| AdSupport.framework | Required | 광고 ID |
| AppTrackingTransparency.framework | Optional | ATT API |
| StoreKit.framework | Required | IAP |

---

## CI/CD 연동

### GitHub Actions 예제
```yaml
# .github/workflows/build.yml
name: Unity Build

on:
  push:
    branches: [main]
  workflow_dispatch:

jobs:
  build-android:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: game-ci/unity-builder@v4
        env:
          UNITY_LICENSE: ${{ secrets.UNITY_LICENSE }}
        with:
          targetPlatform: Android
          buildMethod: BuildAutomation.CLI_BuildAndroid
        env:
          KEYSTORE_PASS: ${{ secrets.KEYSTORE_PASS }}
          KEY_PASS: ${{ secrets.KEY_PASS }}

      - uses: actions/upload-artifact@v4
        with:
          name: android-build
          path: Builds/Android/

  build-ios:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4

      - uses: game-ci/unity-builder@v4
        env:
          UNITY_LICENSE: ${{ secrets.UNITY_LICENSE }}
        with:
          targetPlatform: iOS
          buildMethod: BuildAutomation.CLI_BuildiOS

      - uses: actions/upload-artifact@v4
        with:
          name: ios-xcode-project
          path: Builds/iOS/
```

### Jenkins Pipeline
```groovy
pipeline {
    agent any

    environment {
        KEYSTORE_PASS = credentials('unity-keystore-pass')
        KEY_PASS = credentials('unity-key-pass')
    }

    stages {
        stage('Build Android') {
            steps {
                sh '''
                    Unity -batchmode -quit \
                        -projectPath . \
                        -executeMethod BuildAutomation.CLI_BuildAndroid \
                        -logFile build.log
                '''
            }
        }
    }

    post {
        always {
            archiveArtifacts artifacts: 'Builds/**/*', fingerprint: true
        }
    }
}
```

### 환경 변수

| 변수 | 용도 | 필수 |
|------|------|------|
| KEYSTORE_PASS | Android keystore 비밀번호 | Android 릴리스 |
| KEY_PASS | Key alias 비밀번호 | Android 릴리스 |
| UNITY_LICENSE | Unity 라이선스 | CI/CD |
