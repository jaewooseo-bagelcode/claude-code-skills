# Performance Optimization

GC-free 패턴, Object Pooling, Physics/UI 최적화.

## Table of Contents
1. [GC-free Update 규칙](#gc-free-update-규칙)
2. [Object Pooling](#object-pooling)
3. [Physics 최적화](#physics-최적화)
4. [UI 최적화](#ui-최적화)
5. [Profiling](#profiling)

---

## GC-free Update 규칙

### 금지 목록 (Update/FixedUpdate/LateUpdate)

| 금지 | 이유 | 대안 |
|------|------|------|
| LINQ (.Where, .ToList) | 매 프레임 객체 생성 | for loop |
| 문자열 연결 (+, $"") | GC 유발 | StringBuilder |
| foreach | 일부 컬렉션 boxing | for (int i) |
| Find/GetComponent | 느림 + GC | Awake에서 캐싱 |
| new 키워드 | 힙 할당 | 필드로 재사용 |
| Lambda/Delegate 생성 | 매번 새 객체 | 캐싱된 delegate |

### 올바른 패턴

```csharp
// 필드로 캐싱
private List<Enemy> _filtered = new(100);
private StringBuilder _sb = new(64);
private Rigidbody _rb;
private Transform _player;
private WaitForSeconds _wait = new(1f);

void Awake()
{
    _rb = GetComponent<Rigidbody>();
    _player = GameObject.FindWithTag("Player").transform;
}

void Update()
{
    // for loop
    _filtered.Clear();
    for (int i = 0; i < enemies.Count; i++)
    {
        if (enemies[i].IsActive)
            _filtered.Add(enemies[i]);
    }

    // StringBuilder
    _sb.Clear();
    _sb.Append("Score: ").Append(score);
    scoreText.text = _sb.ToString();

    // 캐싱된 참조
    _rb.AddForce(Vector3.up);
}
```

### 문자열 최적화
```csharp
public class ScoreUI : MonoBehaviour
{
    [SerializeField] private TMP_Text _text;
    private StringBuilder _sb = new(32);
    private int _lastScore = -1;

    public void UpdateScore(int score)
    {
        if (_lastScore == score) return;  // 변경 없으면 스킵
        _lastScore = score;

        _sb.Clear();
        _sb.Append("Score: ").Append(score);
        _text.SetText(_sb);
    }
}
```

---

## Object Pooling

### ObjectSystem (Static API)
```csharp
public static class ObjectSystem
{
    private static ObjectPoolManager Manager => ObjectPoolManager.Instance;

    public static void PreLoad(GameObject prefab, int count) => Manager?.PreLoad(prefab, count);
    public static GameObject Spawn(GameObject prefab, Transform parent = null, bool pooling = true)
        => Manager?.Spawn(prefab, parent, pooling);
    public static void Despawn(Component c) => Manager?.Despawn(c.gameObject);
    public static void Despawn(GameObject obj) => Manager?.Despawn(obj);
}
```

### ObjectPoolManager
```csharp
public class ObjectPoolManager : Singleton<ObjectPoolManager>, IManager
{
    private Dictionary<int, Queue<GameObject>> _pools = new();
    private Dictionary<int, Transform> _containers = new();
    private Transform _root;
    private bool _isInit;

    public bool IsInit => _isInit;

    public void Init()
    {
        if (_root == null)
        {
            _root = new GameObject("[ObjectPool]").transform;
            _root.SetParent(transform);
        }
        _isInit = true;
    }

    public void Clear()
    {
        foreach (var pool in _pools.Values)
        {
            while (pool.Count > 0)
            {
                var obj = pool.Dequeue();
                if (obj != null) Destroy(obj);
            }
        }
        _pools.Clear();
        _containers.Clear();
        _isInit = false;
    }

    public void PreLoad(GameObject prefab, int count)
    {
        int key = prefab.GetInstanceID();
        var pool = GetOrCreatePool(key);
        var container = GetOrCreateContainer(key, prefab.name);

        for (int i = pool.Count; i < count; i++)
            pool.Enqueue(CreateInstance(prefab, container));
    }

    public GameObject Spawn(GameObject prefab, Transform parent = null, bool pooling = true)
    {
        if (!pooling) return Instantiate(prefab, parent);

        int key = prefab.GetInstanceID();
        var pool = GetOrCreatePool(key);
        var container = GetOrCreateContainer(key, prefab.name);

        var obj = pool.Count > 0 ? pool.Dequeue() : CreateInstance(prefab, container);
        obj.transform.SetParent(parent);
        obj.SetActive(true);
        return obj;
    }

    public void Despawn(GameObject obj)
    {
        if (obj == null) return;

        var pooled = obj.GetComponent<PooledObject>();
        if (pooled == null || !_pools.ContainsKey(pooled.PrefabId))
        {
            Destroy(obj);
            return;
        }

        obj.SetActive(false);
        obj.transform.SetParent(_containers[pooled.PrefabId]);
        _pools[pooled.PrefabId].Enqueue(obj);
    }

    private Queue<GameObject> GetOrCreatePool(int key)
    {
        if (!_pools.TryGetValue(key, out var pool))
            _pools[key] = pool = new Queue<GameObject>(32);
        return pool;
    }

    private Transform GetOrCreateContainer(int key, string name)
    {
        if (!_containers.TryGetValue(key, out var container))
        {
            container = new GameObject($"Pool_{name}").transform;
            container.SetParent(_root);
            _containers[key] = container;
        }
        return container;
    }

    private GameObject CreateInstance(GameObject prefab, Transform container)
    {
        var obj = Instantiate(prefab, container);
        obj.SetActive(false);
        obj.AddComponent<PooledObject>().PrefabId = prefab.GetInstanceID();
        return obj;
    }
}

public class PooledObject : MonoBehaviour { public int PrefabId; }
```

### 자동 반환
```csharp
public class AutoDespawn : MonoBehaviour
{
    [SerializeField] private float _lifetime = 2f;
    private float _timer;

    void OnEnable() => _timer = _lifetime;

    void Update()
    {
        _timer -= Time.deltaTime;
        if (_timer <= 0) ObjectSystem.Despawn(gameObject);
    }
}
```

---

## Physics 최적화

### NonAlloc 메서드
```csharp
public static class PhysicsHelper
{
    private static RaycastHit[] _hits = new RaycastHit[32];
    private static Collider[] _colliders = new Collider[64];

    public static int Raycast(Vector3 origin, Vector3 dir, float dist, int layer)
        => Physics.RaycastNonAlloc(origin, dir, _hits, dist, layer);

    public static int OverlapSphere(Vector3 pos, float radius, int layer)
        => Physics.OverlapSphereNonAlloc(pos, radius, _colliders, layer);

    public static RaycastHit GetHit(int i) => _hits[i];
    public static Collider GetCollider(int i) => _colliders[i];
}

// 사용
void FindNearby()
{
    int count = PhysicsHelper.OverlapSphere(transform.position, 10f, enemyLayer);
    for (int i = 0; i < count; i++)
    {
        var col = PhysicsHelper.GetCollider(i);
        // process
    }
}
```

### CompareTag 필수
```csharp
// ❌ 느림 + GC
if (other.tag == "Enemy")

// ✅ 빠름
if (other.CompareTag("Enemy"))
```

### Physics Settings
```
Project Settings > Physics/Time:
- Fixed Timestep: 0.02 (기본값, 50fps 물리)
  - 성능 이슈 시 0.03으로 조정 가능
- Default Solver Iterations: 6 → 4
- Layer Collision Matrix: 불필요한 충돌 비활성화
```

> **Note**: Fixed Timestep 조정 시 물리 정확도와 성능 트레이드오프 고려.

---

## UI 최적화

### Canvas 분리
```
UI/
├── StaticCanvas     → 변하지 않는 UI (배경, 로고)
├── DynamicCanvas    → 자주 변하는 UI (점수, HP)
└── PopupCanvas      → 팝업
```

### Raycast Target
클릭 불필요한 UI 요소는 Raycast Target = false

```csharp
#if UNITY_EDITOR
[MenuItem("Tools/Disable Raycast Targets")]
static void DisableRaycastTargets()
{
    foreach (var g in FindObjectsOfType<Graphic>())
    {
        if (g.GetComponent<Button>() == null &&
            g.GetComponent<Toggle>() == null)
            g.raycastTarget = false;
    }
}
#endif
```

---

## Profiling

### 목표 수치

| 항목 | 목표 |
|------|------|
| GC Alloc | 0KB/frame |
| Frame Time | < 16.6ms (60fps) |
| Draw Calls | < 100 |
| Memory | < 500MB |

### ProfilerMarker
```csharp
using UnityEngine.Profiling;

public class GameSystem : MonoBehaviour
{
    private static readonly ProfilerMarker _marker = new("GameSystem.Update");

    void Update()
    {
        _marker.Begin();
        // 측정할 코드
        _marker.End();
    }
}
```
