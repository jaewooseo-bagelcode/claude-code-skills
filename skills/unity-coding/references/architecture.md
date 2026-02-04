# Architecture Patterns

System + Manager 패턴, Singleton, GameManager 구조, 실전 예제.

## Table of Contents
1. [System + Manager 패턴](#system--manager-패턴)
2. [Singleton 베이스 클래스](#singleton-베이스-클래스)
3. [GameManager 구조](#gamemanager-구조)
4. [실전 예제](#실전-예제)

**See also:**
- [Event Patterns](architecture-events.md) - GameEvents, GameEventManager
- [Advanced Patterns](architecture-advanced.md) - MonoBase, Component Pool, Async FSM, Disposable System, EventNotifier

---

## System + Manager 패턴

### 구조
```
┌─────────────────┐
│  외부 코드       │  ProjectileSystem.Spawn(id, data);
└────────┬────────┘
         │ (static 호출)
         ▼
┌─────────────────┐
│  System         │  public static class ProjectileSystem
│  (Static API)   │  - 깔끔한 API 제공
│                 │  - null 체크 내장
└────────┬────────┘
         │ (Manager?.Method())
         ▼
┌─────────────────┐
│  Manager        │  public class ProjectileManager : Singleton<T>
│  (Singleton)    │  - 실제 로직 구현
│                 │  - 상태 관리
└─────────────────┘
```

### System 템플릿
```csharp
public static class {Domain}System
{
    private static {Domain}Manager Manager => {Domain}Manager.Instance;

    public static bool IsInit => Manager?.IsInit ?? false;
    public static int ActiveCount => Manager?.ActiveCount ?? 0;

    public static void Init() => Manager?.Init();
    public static void Clear() => Manager?.Clear();
    public static void PreLoad(int id, int count) => Manager?.PreLoad(id, count);
    public static {Type} Spawn(int id, SpawnData data) => Manager?.Spawn(id, data);
    public static void Despawn({Type} item) => Manager?.Despawn(item);
}
```

### Manager 템플릿
```csharp
public class {Domain}Manager : Singleton<{Domain}Manager>, IManager
{
    [SerializeField] private {Domain}Asset _asset;

    private List<{Type}> _activeList = new();
    private bool _isInit;

    public bool IsInit => _isInit;
    public int ActiveCount => _activeList.Count;
    public IReadOnlyList<{Type}> ActiveList => _activeList;

    public void Init()
    {
        _activeList.Clear();
        _isInit = true;
    }

    public void Clear()
    {
        DespawnAll();
        _isInit = false;
    }

    public void PreLoad(int id, int count)
    {
        var prefab = _asset.GetPrefab(id);
        if (prefab == null) return;
        ObjectSystem.PreLoad(prefab, count);
    }

    public {Type} Spawn(int id, SpawnData data)
    {
        var prefab = _asset.GetPrefab(id);
        if (prefab == null) return null;

        var obj = ObjectSystem.Spawn(prefab, pooling: true);
        var item = obj.GetComponent<{Type}>();
        item.Init(data);
        _activeList.Add(item);
        return item;
    }

    public void Despawn({Type} item)
    {
        if (item == null || !_activeList.Contains(item)) return;
        _activeList.Remove(item);
        ObjectSystem.Despawn(item);
    }

    public void DespawnAll()
    {
        for (int i = _activeList.Count - 1; i >= 0; i--)
        {
            if (_activeList[i] != null)
                ObjectSystem.Despawn(_activeList[i]);
        }
        _activeList.Clear();
    }
}

public interface IManager
{
    bool IsInit { get; }
    void Init();
    void Clear();
}

// 확장 인터페이스: 생명주기 콜백 (선택적)
public interface IManagerLifecycle : IManager
{
    void OnAwake();    // Globals.Awake에서 호출
    void OnDestroy();  // Globals.OnDestroy에서 호출
}
```

---

## Singleton 베이스 클래스

### 전역 Singleton (DontDestroyOnLoad)
```csharp
public abstract class Singleton<T> : MonoBehaviour where T : MonoBehaviour
{
    private static T _instance;
    private static readonly object _lock = new();
    private static bool _applicationIsQuitting;

    public static T Instance
    {
        get
        {
            if (_applicationIsQuitting) return null;

            lock (_lock)
            {
                if (_instance == null)
                {
                    _instance = FindFirstObjectByType<T>();
                    if (_instance == null)
                    {
                        var go = new GameObject($"[{typeof(T).Name}]");
                        _instance = go.AddComponent<T>();
                    }
                }
                return _instance;
            }
        }
    }

    protected virtual void Awake()
    {
        if (_instance != null && _instance != this)
        {
            Destroy(gameObject);
            return;
        }
        _instance = this as T;
        DontDestroyOnLoad(gameObject);
        OnAwake();
    }

    protected virtual void OnAwake() { }
    protected virtual void OnApplicationQuit() => _applicationIsQuitting = true;
}
```

### 씬 단위 Singleton
```csharp
public abstract class SingletonWithScene<T> : MonoBehaviour where T : MonoBehaviour
{
    public static T Instance { get; private set; }

    protected virtual void Awake()
    {
        Instance = this as T;
        OnAwake();
    }

    protected virtual void OnAwake() { }

    protected virtual void OnDestroy()
    {
        if (Instance == this) Instance = null;
    }
}
```

---

## GameManager 구조

### BaseGameManager
```csharp
public abstract class BaseGameManager : Singleton<BaseGameManager>
{
    protected List<IManager> preLoadManagerList = new();
    protected List<IManager> managerList = new();

    protected override void OnAwake()
    {
        AddPreLoadManagers();
        AddManagers();
        StartCoroutine(InitializeManagers());
    }

    protected abstract void AddPreLoadManagers();
    protected abstract void AddManagers();
    protected virtual void OnPreLoad() { }
    protected virtual void OnInit() { }

    private IEnumerator InitializeManagers()
    {
        foreach (var manager in preLoadManagerList)
        {
            manager.Init();
            yield return null;
        }
        OnPreLoad();

        foreach (var manager in managerList)
        {
            manager.Init();
            yield return null;
        }
        OnInit();
    }
}
```

### GameManager 구현
```csharp
public class GameManager : BaseGameManager
{
    protected override void AddPreLoadManagers()
    {
        preLoadManagerList.Add(UIManager.Instance);
        preLoadManagerList.Add(SceneFlowManager.Instance);
    }

    protected override void AddManagers()
    {
        // 순서 중요!
        managerList.Add(ObjectPoolManager.Instance);
        managerList.Add(AudioManager.Instance);
        managerList.Add(PlayerManager.Instance);
        managerList.Add(EnemyManager.Instance);
    }

    protected override void OnInit()
    {
        Application.targetFrameRate = 60;
    }
}
```

### BaseSceneManager
```csharp
public abstract class BaseSceneManager<T> : SingletonWithScene<T> where T : MonoBehaviour
{
    protected List<IManager> managerList = new();

    protected override void OnAwake()
    {
        base.OnAwake();
        AddManagers();
        StartCoroutine(InitializeManagers());
    }

    protected abstract void AddManagers();
    protected virtual void OnInit() { }
    protected virtual void OnClear() { }

    private IEnumerator InitializeManagers()
    {
        foreach (var manager in managerList)
        {
            manager.Init();
            yield return null;
        }
        OnInit();
    }

    protected virtual void OnDestroy()
    {
        base.OnDestroy();
        OnClear();
    }
}
```

---

## 실전 예제

### EnemySystem + EnemyManager
```csharp
// ===== EnemySystem.cs =====
public static class EnemySystem
{
    private static EnemyManager Manager => EnemyManager.Instance;

    public static bool IsInit => Manager?.IsInit ?? false;
    public static int ActiveCount => Manager?.ActiveCount ?? 0;

    public static void PreLoad(EnemyType type, int count) => Manager?.PreLoad(type, count);
    public static Enemy Spawn(EnemyType type, Vector3 pos) => Manager?.Spawn(type, pos);
    public static void Despawn(Enemy e) => Manager?.Despawn(e);
    public static void DespawnAll() => Manager?.DespawnAll();
}

// ===== EnemyManager.cs =====
public class EnemyManager : SingletonWithScene<EnemyManager>, IManager
{
    [SerializeField] private EnemyAsset _asset;
    [SerializeField] private Transform _container;

    private List<Enemy> _activeList = new(50);
    private bool _isInit;

    public bool IsInit => _isInit;
    public int ActiveCount => _activeList.Count;

    public void Init() { _activeList.Clear(); _isInit = true; }
    public void Clear() { DespawnAll(); _isInit = false; }

    public void PreLoad(EnemyType type, int count)
    {
        ObjectSystem.PreLoad(_asset.GetPrefab(type), count);
    }

    public Enemy Spawn(EnemyType type, Vector3 position)
    {
        var obj = ObjectSystem.Spawn(_asset.GetPrefab(type), _container, pooling: true);
        var enemy = obj.GetComponent<Enemy>();
        enemy.transform.position = position;
        enemy.Init(_asset.GetData(type));
        enemy.OnDeath += HandleDeath;
        _activeList.Add(enemy);
        return enemy;
    }

    public void Despawn(Enemy enemy)
    {
        if (!_activeList.Contains(enemy)) return;
        enemy.OnDeath -= HandleDeath;
        _activeList.Remove(enemy);
        ObjectSystem.Despawn(enemy);
    }

    public void DespawnAll()
    {
        for (int i = _activeList.Count - 1; i >= 0; i--)
        {
            _activeList[i].OnDeath -= HandleDeath;
            ObjectSystem.Despawn(_activeList[i]);
        }
        _activeList.Clear();
    }

    private void HandleDeath(Enemy e) => Despawn(e);
}

// ===== EnemyAsset.cs =====
[CreateAssetMenu(menuName = "Game/Enemy Asset")]
public class EnemyAsset : ScriptableObject
{
    [System.Serializable]
    public class Entry { public EnemyType type; public GameObject prefab; public EnemyData data; }

    [SerializeField] private List<Entry> _enemies;

    public GameObject GetPrefab(EnemyType type) => _enemies.Find(e => e.type == type)?.prefab;
    public EnemyData GetData(EnemyType type) => _enemies.Find(e => e.type == type)?.data;
}
```
