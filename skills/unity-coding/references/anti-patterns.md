# Anti-Patterns

코드 리뷰 체크리스트. 상세 내용은 각 참조 파일 확인.

## 코드 리뷰 체크리스트

### 성능 (→ [performance.md](performance.md))
- [ ] Update에서 Find/GetComponent 없음
- [ ] Update에서 LINQ 없음
- [ ] Update에서 문자열 연결 없음
- [ ] Update에서 new 키워드 없음
- [ ] CompareTag() 사용
- [ ] Object Pooling 사용 (Instantiate/Destroy 금지)
- [ ] NonAlloc Physics 메서드 사용

### 아키텍처 (→ [architecture.md](architecture.md))
- [ ] God Class 없음 (< 500줄)
- [ ] 순환 참조 없음
- [ ] 상속 깊이 < 3단계
- [ ] System + Manager 패턴 사용

### 메모리
- [ ] 이벤트 구독/해제 쌍 (OnEnable/OnDisable)
- [ ] Coroutine OnDisable에서 정리
- [ ] Resources 최소 사용

### 빌드 (→ [build-optimization.md](build-optimization.md))
- [ ] 텍스처 ASTC 6x6 압축
- [ ] 오디오 Vorbis 압축
- [ ] IL2CPP + Managed Stripping High

---

## 빠른 참조: 흔한 실수

### 이벤트 누수
```csharp
// ❌ OnDisable 없음
void OnEnable() { GameEvents.OnDeath += Handle; }

// ✅ 쌍으로
void OnEnable() { GameEvents.OnDeath += Handle; }
void OnDisable() { GameEvents.OnDeath -= Handle; }
```

### Coroutine 정리
```csharp
// ❌ 정리 안함
void Start() { StartCoroutine(Loop()); }

// ✅ 정리
Coroutine _routine;
void Start() { _routine = StartCoroutine(Loop()); }
void OnDisable() { if (_routine != null) StopCoroutine(_routine); }
```

### 순환 참조
```csharp
// ❌ A → B → A
class EnemyManager { PlayerManager player; }
class PlayerManager { EnemyManager enemy; }

// ✅ 이벤트 사용
class EnemyManager { public event Action OnDeath; }
class PlayerManager { void Start() { EnemyManager.Instance.OnDeath += Handle; } }
```

### God Class
```csharp
// ❌ 모든 것을 하나에
class Player : MonoBehaviour { /* 2000줄 */ }

// ✅ 분리
class PlayerMovement : MonoBehaviour { }
class PlayerCombat : MonoBehaviour { }
class PlayerInventory : MonoBehaviour { }
```

### 매직 넘버
```csharp
// ❌
if (health < 30) ShowWarning();

// ✅
const float LOW_HEALTH = 30f;
if (health < LOW_HEALTH) ShowWarning();
```

### public 필드
```csharp
// ❌ 어디서든 수정 가능
public int health;

// ✅ 캡슐화
[SerializeField] private int _health;
public int Health => _health;
public void TakeDamage(int dmg) { _health -= dmg; }
```

### 빈 MonoBehaviour 메서드
```csharp
// ❌ 불필요한 호출 오버헤드
void Start() { }
void Update() { }

// ✅ 삭제
```

### DI 프레임워크 (하이퍼캐주얼)
```csharp
// ❌ 오버킬
[Inject] IPlayerService _player;

// ✅ 단순하게
PlayerSystem.AddScore(100);
```
