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

---

## Unity Object Null 체크

Unity 오브젝트는 C# 표준 null과 다르게 동작. `Destroy()` 후에도 C# 참조는 살아있지만 Unity는 null로 취급.

### 올바른 패턴
```csharp
GameObject target;
Destroy(target);

// 다음 프레임:
// ✅ 올바른 체크 - Unity의 == 오버라이드 사용
if (!target) { }           // implicit bool (권장)
if (target == null) { }    // Unity == 연산자
if (target != null) { }

// ❌ 위험 - 파괴된 오브젝트 감지 못함
if (target is null) { }              // 패턴 매칭 우회
if (target is not null) { }
if (ReferenceEquals(target, null)) { }
```

### 인터페이스 캐스팅 주의
```csharp
// ❌ 위험 - Unity == 보호 상실
IMyInterface iface = GetComponent<MyComponent>();
if (iface == null) { }  // Destroy 후에도 false!

// ✅ 명시적 캐스팅
if ((UnityEngine.Object)iface == null) { }

// ✅ 더 나은 방법: TryGetComponent (Unity 2019.2+)
if (TryGetComponent<MyComponent>(out var comp))
{
    // comp 사용
}
```

### TryGetComponent 권장
```csharp
// ❌ 구식
var rb = GetComponent<Rigidbody>();
if (rb != null)
    rb.AddForce(Vector3.up);

// ✅ 현대적 - null 체크 + GetComponent 한 번에
if (TryGetComponent<Rigidbody>(out var rb))
    rb.AddForce(Vector3.up);
```

**체크리스트 추가**:
- [ ] `is null` 패턴 매칭 사용 안 함 (Unity 오브젝트)
- [ ] TryGetComponent 패턴 사용

---

## Input System (New) - Unity 6+

Unity 6부터 New Input System이 기본. 레거시 `Input.GetKey()` deprecated.

### 레거시 vs New
```csharp
// ❌ 레거시 (deprecated in Unity 6)
void Update()
{
    if (Input.GetKeyDown(KeyCode.Space))
        Jump();

    float h = Input.GetAxis("Horizontal");
}

// ✅ New Input System
using UnityEngine.InputSystem;

public class PlayerController : MonoBehaviour
{
    private InputAction _move;
    private InputAction _jump;

    void Awake()
    {
        _move = new InputAction("Move", binding: "<Gamepad>/leftStick");
        _move.AddCompositeBinding("2DVector")
            .With("Up", "<Keyboard>/w")
            .With("Down", "<Keyboard>/s")
            .With("Left", "<Keyboard>/a")
            .With("Right", "<Keyboard>/d");

        _jump = new InputAction("Jump", binding: "<Keyboard>/space");
    }

    void OnEnable()
    {
        _move.Enable();
        _jump.Enable();
        _jump.performed += OnJump;  // 구독
    }

    void OnDisable()
    {
        _jump.performed -= OnJump;  // 해제 필수!
        _move.Disable();
        _jump.Disable();
    }

    void Update()
    {
        Vector2 input = _move.ReadValue<Vector2>();
        // 이동 처리
    }

    private void OnJump(InputAction.CallbackContext ctx) => Jump();
}
```

### InputAction 타입
| 타입 | 용도 | 예시 |
|------|------|------|
| Button | 단일 입력 | 점프, 공격 |
| Value | 연속 입력 | 이동 (Vector2), 시점 |
| Pass-Through | 모든 변화 | 디버그, 녹화 |

### 핵심 규칙
- **OnEnable/OnDisable 쌍**: 이벤트 구독 관리 필수 (풀링 시 특히 중요)
- **Enable() 호출**: InputAction은 활성화 후에만 동작
- **Player Settings 확인**: Active Input Handling = "Input System Package (New)"

**체크리스트 추가**:
- [ ] New Input System 사용 (레거시 Input 금지)
- [ ] InputAction OnEnable/OnDisable 쌍 관리
