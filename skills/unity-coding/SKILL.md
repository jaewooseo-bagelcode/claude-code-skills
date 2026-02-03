---
name: unity-coding
description: >
  하이퍼캐주얼 게임 개발 전문 스킬. Voodoo, Habi, ABI 스타일 1-2주 프로토타입에 최적화.
  DI 프레임워크 없이 System+Manager 패턴, ObjectSystem 기반 풀링, GC-free Update 패턴 제공.
  모바일 빌드 사이즈 최적화(ASTC, Strip, Managed Stripping) 포함.
  사용 시점 - (1) 하이퍼캐주얼 게임 개발, (2) 빠른 프로토타입 제작,
  (3) 모바일 성능/빌드 사이즈 최적화, (4) CPI 테스트용 게임 제작,
  (5) Unity, hypercasual, 하이퍼캐주얼, 프로토타입, CPI 키워드 언급 시
---

# Unity Coding (Hypercasual)

**1-2주 프로토타입** | **DI 프레임워크 없음** | **극한의 최적화**

## Core Philosophy

| 사용하지 않음 | 사용함 |
|--------------|--------|
| VContainer, Zenject, SignalBus | System + Manager 패턴 |
| 복잡한 아키텍처 | 단순한 Singleton 구조 |
| Instantiate/Destroy | ObjectSystem 풀링 |
| LINQ, 문자열 연결 (Update) | for loop, StringBuilder |

## Quick Reference

### System + Manager 패턴
```
외부 코드 → [System (static)] → [Manager (Singleton)]
           ProjectileSystem.Spawn()   ProjectileManager.Instance
```
- System: static class, null-safe API 제공
- Manager: Singleton, 실제 로직 구현

### 필수 규칙
- **Update 금지**: Find, GetComponent, LINQ, 문자열 연결, new
- **풀링 필수**: `ObjectSystem.Spawn()` / `ObjectSystem.Despawn()`
- **태그 비교**: `CompareTag()` 사용 (tag == "string" 금지)
- **Physics**: NonAlloc 메서드 사용

### 빌드 목표
| 단계 | 사이즈 |
|------|--------|
| CPI 테스트 | < 80MB |
| 소프트런칭 | < 150MB |
| 글로벌 런칭 | < 200MB |

### 폴더 구조
```
Assets/__GameName/
├── _Objects/{Player,Enemy,Projectile}/Scripts/
├── Scripts/_Core/{GameManager,ObjectManager}/
├── Prefabs/
└── ScriptableObjects/
```

## When to Read References

| 상황 | 참조 파일 |
|------|----------|
| 새 System/Manager 구현 | [architecture.md](references/architecture.md) |
| 성능 문제, GC 스파이크 | [performance.md](references/performance.md) |
| 빌드 사이즈 줄이기 | [build-optimization.md](references/build-optimization.md) |
| 코드 리뷰 체크리스트 | [anti-patterns.md](references/anti-patterns.md) |
