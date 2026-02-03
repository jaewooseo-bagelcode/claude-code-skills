# Build Optimization

빌드 사이즈, Player Settings, 에셋 압축, Code Stripping.

## Table of Contents
1. [빌드 사이즈 목표](#빌드-사이즈-목표)
2. [Player Settings](#player-settings)
3. [Texture 압축](#texture-압축)
4. [Audio 압축](#audio-압축)
5. [Code Stripping](#code-stripping)
6. [에디터 스크립트](#에디터-스크립트)

---

## 빌드 사이즈 목표

| 단계 | 목표 | 비고 |
|------|------|------|
| CPI 테스트 | < 250MB | WiFi 없이 다운로드 가능 (iOS 기준) |
| 소프트런칭 | < 300MB | 기능 추가 허용 |
| 글로벌 런칭 | < 400MB | 최대 허용치 |

> **Note**: 낮을수록 다운로드 전환율 유리. 250MB는 WiFi 없이 다운로드 가능한 임계치.

---

## Player Settings

### Android
```
Resolution and Presentation:
- Default Orientation: Portrait

Other Settings:
- Scripting Backend: IL2CPP (필수)
- Api Compatibility Level: .NET Standard 2.1
- Target Architectures: ARM64 (ARMv7 제외)

Optimization:
- Managed Stripping Level: High
- Strip Engine Code: true
- Vertex Compression: Everything
- Optimize Mesh Data: true
```

### iOS
```
Other Settings:
- Scripting Backend: IL2CPP
- Architecture: ARM64

Optimization:
- Managed Stripping Level: High
- Strip Engine Code: true
- Script Call Optimization: Fast but no Exceptions
```

### Quality Settings
```
모바일 Quality Level:
- Pixel Light Count: 1
- Anisotropic Textures: Disabled
- Anti Aliasing: Disabled
- Soft Particles: Disabled
- Shadows: Hard Shadows Only 또는 Disable
- Shadow Resolution: Low
```

---

## Texture 압축

### 권장 설정
| 타입 | Max Size | Format | Mip Maps | 비고 |
|------|----------|--------|----------|------|
| 일반 텍스처 | 2048 | ASTC 6x6 | 3D만 | 배칭/패킹 최적화에 유리 |
| UI 스프라이트 | 1024 | ASTC 6x6 | Off | |
| Normal Map | 1024 | ASTC 6x6 | On | |
| 고품질 텍스처 | 4096 | ASTC 6x6 | On | 하이엔드 기기 타겟 |

**공통**: Read/Write Enabled = false

> **Note**: 큰 텍스처가 배칭/패킹 최적화에 유리. 아시아 시장 고려 시 2048 안정적, 하이엔드는 4096도 가능.

### Sprite Atlas
```
Settings:
- Allow Rotation: true
- Tight Packing: true
- Padding: 2

Platform Override:
- Max Texture Size: 4096 (또는 2048)
- Format: ASTC 6x6
```

---

## Audio 압축

### 권장 설정
| 타입 | Load Type | Format | Quality | Sample Rate |
|------|-----------|--------|---------|-------------|
| 효과음 (< 5초) | Decompress on Load | Vorbis | 70% | 22050 Hz |
| BGM (> 5초) | Streaming | Vorbis | 50% | 44100 Hz |

**효과음**: Force To Mono = true

---

## Code Stripping

### Managed Stripping Level
```
High (권장) - 최대 스트리핑
Strip Engine Code: true
```

### link.xml (스트리핑 예외)
Reflection 사용 시 필요:

```xml
<!-- Assets/link.xml -->
<linker>
    <!-- 특정 타입 보호 -->
    <assembly fullname="Assembly-CSharp">
        <type fullname="MyGame.SaveData" preserve="all"/>
    </assembly>

    <!-- SDK 보호 -->
    <assembly fullname="Firebase.App" preserve="all"/>
    <assembly fullname="Firebase.Analytics" preserve="all"/>
</linker>
```

---

## 에디터 스크립트

### 텍스처 일괄 최적화
```csharp
#if UNITY_EDITOR
using UnityEditor;

public class AssetOptimizer
{
    [MenuItem("Tools/Optimize Textures")]
    static void OptimizeTextures()
    {
        foreach (var guid in AssetDatabase.FindAssets("t:Texture2D"))
        {
            var path = AssetDatabase.GUIDToAssetPath(guid);
            if (path.Contains("/Plugins/")) continue;

            var importer = AssetImporter.GetAtPath(path) as TextureImporter;
            if (importer == null) continue;

            bool changed = false;

            if (importer.maxTextureSize > 2048)
            { importer.maxTextureSize = 2048; changed = true; }

            if (importer.isReadable)
            { importer.isReadable = false; changed = true; }

            // ASTC 6x6
            var android = importer.GetPlatformTextureSettings("Android");
            if (android.format != TextureImporterFormat.ASTC_6x6)
            {
                android.overridden = true;
                android.format = TextureImporterFormat.ASTC_6x6;
                importer.SetPlatformTextureSettings(android);
                changed = true;
            }

            var ios = importer.GetPlatformTextureSettings("iPhone");
            if (ios.format != TextureImporterFormat.ASTC_6x6)
            {
                ios.overridden = true;
                ios.format = TextureImporterFormat.ASTC_6x6;
                importer.SetPlatformTextureSettings(ios);
                changed = true;
            }

            if (changed) importer.SaveAndReimport();
        }
    }

    [MenuItem("Tools/Optimize Audio")]
    static void OptimizeAudio()
    {
        foreach (var guid in AssetDatabase.FindAssets("t:AudioClip"))
        {
            var path = AssetDatabase.GUIDToAssetPath(guid);
            var importer = AssetImporter.GetAtPath(path) as AudioImporter;
            if (importer == null) continue;

            var clip = AssetDatabase.LoadAssetAtPath<AudioClip>(path);
            bool isShort = clip.length < 5f;

            var settings = importer.defaultSampleSettings;
            settings.loadType = isShort ? AudioClipLoadType.DecompressOnLoad : AudioClipLoadType.Streaming;
            settings.compressionFormat = AudioCompressionFormat.Vorbis;
            settings.quality = isShort ? 0.7f : 0.5f;

            if (isShort)
            {
                settings.sampleRateSetting = AudioSampleRateSetting.OverrideSampleRate;
                settings.sampleRateOverride = 22050;
            }

            importer.defaultSampleSettings = settings;
            importer.forceToMono = isShort;
            importer.SaveAndReimport();
        }
    }
}
#endif
```

### 빌드 리포트
```csharp
#if UNITY_EDITOR
using UnityEditor.Build.Reporting;

public class BuildReporter : IPostprocessBuildWithReport
{
    public int callbackOrder => 0;

    public void OnPostprocessBuild(BuildReport report)
    {
        Debug.Log($"Build: {report.summary.result}");
        Debug.Log($"Size: {report.summary.totalSize / 1024 / 1024} MB");

        var files = report.GetFiles();
        System.Array.Sort(files, (a, b) => b.size.CompareTo(a.size));

        Debug.Log("=== Top 10 Largest Files ===");
        for (int i = 0; i < Mathf.Min(10, files.Length); i++)
            Debug.Log($"{files[i].size / 1024}KB - {files[i].path}");
    }
}
#endif
```
