# Architecture Patterns

System + Manager 패턴, Singleton, GameManager 구조, 이벤트 통신, 고급 패턴.

## Table of Contents
1. [System + Manager 패턴](#system--manager-패턴)
2. [Singleton 베이스 클래스](#singleton-베이스-클래스)
3. [GameManager 구조](#gamemanager-구조)
4. [실전 예제](#실전-예제)
5. [GameEvents (C# Static Events)](#gameevents-c-static-events) - 코드 기반 이벤트 버스
6. [Advanced: MonoBase](#advanced-monobase) - IDisposable 자동 관리
7. [Advanced: Component Pool](#advanced-component-pool) - 컴포넌트 기반 풀링
8. [Advanced: Async FSM](#advanced-async-fsm) - UniTask 기반 상태 머신

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

---

## GameEvents (C# Static Events)

ScriptableObject 없이 순수 코드로 이벤트 버스 구현. AI 친화적.

### 장점
- ScriptableObject 에셋 생성 불필요
- IDE 자동완성, 리팩토링 완전 지원
- 컴파일 타임 검증
- 명확한 스택 트레이스

### GameEvents 클래스
```csharp
using System;
using UnityEngine;

public static class GameEvents
{
    // Game state events
    public static event Action OnGameStarted;
    public static event Action OnGamePaused;
    public static event Action OnGameResumed;
    public static event Action OnGameOver;

    // Player events
    public static event Action OnPlayerDied;
    public static event Action OnPlayerRespawned;
    public static event Action<float> OnHealthChanged;  // normalized 0-1

    // Score/Progress events
    public static event Action<int> OnScoreChanged;
    public static event Action OnLevelComplete;
    public static event Action<int> OnLevelStarted;  // level index

    // UI events
    public static event Action<string> OnShowPopup;

    // Raise methods (encapsulation)
    public static void GameStarted() => OnGameStarted?.Invoke();
    public static void GamePaused() => OnGamePaused?.Invoke();
    public static void GameResumed() => OnGameResumed?.Invoke();
    public static void GameOver() => OnGameOver?.Invoke();

    public static void PlayerDied() => OnPlayerDied?.Invoke();
    public static void PlayerRespawned() => OnPlayerRespawned?.Invoke();
    public static void HealthChanged(float normalizedHealth) => OnHealthChanged?.Invoke(normalizedHealth);

    public static void ScoreChanged(int score) => OnScoreChanged?.Invoke(score);
    public static void LevelComplete() => OnLevelComplete?.Invoke();
    public static void LevelStarted(int levelIndex) => OnLevelStarted?.Invoke(levelIndex);

    public static void ShowPopup(string message) => OnShowPopup?.Invoke(message);

    /// <summary>
    /// Reset all events - Domain Reload 대응
    /// </summary>
    [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.SubsystemRegistration)]
    public static void ResetAll()
    {
        OnGameStarted = null;
        OnGamePaused = null;
        OnGameResumed = null;
        OnGameOver = null;
        OnPlayerDied = null;
        OnPlayerRespawned = null;
        OnHealthChanged = null;
        OnScoreChanged = null;
        OnLevelComplete = null;
        OnLevelStarted = null;
        OnShowPopup = null;
    }
}
```

### 사용 예
```csharp
// ===== Publisher (이벤트 발행) =====
public class Player : MonoBehaviour
{
    void Die()
    {
        GameEvents.PlayerDied();  // 한 줄로 끝
    }

    void TakeDamage(float damage)
    {
        currentHealth -= damage;
        GameEvents.HealthChanged(currentHealth / maxHealth);

        if (currentHealth <= 0)
            Die();
    }
}

// ===== Subscriber (이벤트 구독) =====
public class GameOverUI : MonoBehaviour
{
    [SerializeField] private GameObject gameOverPanel;

    void OnEnable()
    {
        GameEvents.OnPlayerDied += ShowGameOver;
    }

    void OnDisable()
    {
        GameEvents.OnPlayerDied -= ShowGameOver;  // 반드시 해제!
    }

    private void ShowGameOver()
    {
        gameOverPanel.SetActive(true);
    }
}

// ===== 멀티 구독 예 (AudioManager) =====
public class AudioManager : MonoBehaviour
{
    void OnEnable()
    {
        GameEvents.OnPlayerDied += PlayDeathSound;
        GameEvents.OnLevelComplete += PlayVictorySound;
        GameEvents.OnScoreChanged += _ => PlaySound("Score");
    }

    void OnDisable()
    {
        GameEvents.OnPlayerDied -= PlayDeathSound;
        GameEvents.OnLevelComplete -= PlayVictorySound;
    }

    private void PlayDeathSound() => PlaySound("Death");
    private void PlayVictorySound() => PlaySound("Victory");
}
```

### System+Manager vs GameEvents

| 용도 | 권장 패턴 |
|------|----------|
| 오브젝트 생명주기 (Spawn/Despawn) | System + Manager |
| 상태 변화 알림 (Score, Health) | GameEvents |
| 양방향 통신 | System + Manager |
| 단방향 브로드캐스트 | GameEvents |

> **Note**: 둘 다 같이 사용해도 됨. System+Manager는 "관리", GameEvents는 "알림" 용도로 구분.

---

## Advanced: MonoBase

프로젝트가 커지면 IDisposable 관리와 CancellationToken 처리가 중요해짐.

### MonoBase - IDisposable 자동 정리
```csharp
using System;
using System.Collections.Generic;
using System.Threading;

public abstract class MonoBase : MonoBehaviour, ICollection<IDisposable>
{
    private readonly List<IDisposable> _disposables = new();
    protected CancellationTokenSource DisableCts { get; private set; }

    public int Count => _disposables.Count;
    public bool IsReadOnly => false;

    protected virtual void OnEnable()
    {
        DisableCts?.Cancel();
        DisableCts?.Dispose();
        DisableCts = new CancellationTokenSource();
    }

    protected virtual void OnDisable()
    {
        DisableCts?.Cancel();
        DisableCts?.Dispose();
        DisableCts = null;
    }

    protected virtual void OnDestroy()
    {
        foreach (var d in _disposables)
            d?.Dispose();
        _disposables.Clear();
    }

    // ICollection<IDisposable>
    public void Add(IDisposable item) => _disposables.Add(item);
    public void Clear() => _disposables.Clear();
    public bool Contains(IDisposable item) => _disposables.Contains(item);
    public void CopyTo(IDisposable[] array, int index) => _disposables.CopyTo(array, index);
    public bool Remove(IDisposable item) => _disposables.Remove(item);
    public IEnumerator<IDisposable> GetEnumerator() => _disposables.GetEnumerator();
    System.Collections.IEnumerator System.Collections.IEnumerable.GetEnumerator() => GetEnumerator();
}
```

### 사용 예
```csharp
public class PlayerController : MonoBase
{
    protected override void OnEnable()
    {
        base.OnEnable();

        // Rx 구독 자동 정리
        Add(GameEvents.OnScoreChanged.Subscribe(OnScoreChanged));

        // 이벤트 구독 (IDisposable 래퍼)
        Add(Disposable.Create(() => SomeEvent -= Handler));
        SomeEvent += Handler;
    }

    async void StartAsync()
    {
        // OnDisable 시 자동 취소
        await SomeAsyncOperation(DisableCts.Token);
    }
}
```

---

## Advanced: Component Pool

각 프리팹별로 ObjectPool 컴포넌트를 붙이는 방식. 인스펙터에서 설정 가능.

### ObjectPool (컴포넌트)
```csharp
public class ObjectPool : MonoBehaviour
{
    [SerializeField] private GameObject _prefab;
    [SerializeField] private int _poolSize = 10;

    private List<PooledObject> _available = new();

    void Awake() => CreatePool();

    public void CreatePool()
    {
        while (_available.Count < _poolSize)
            _available.Add(CreateObject());
    }

    public PooledObject Get(bool activate = true)
    {
        PooledObject obj;
        if (_available.Count > 0)
        {
            obj = _available[^1];
            _available.RemoveAt(_available.Count - 1);
        }
        else
        {
            obj = CreateObject();
        }

        if (activate) obj.gameObject.SetActive(true);
        return obj;
    }

    public void Return(PooledObject obj)
    {
        obj.gameObject.SetActive(false);
        obj.transform.SetParent(transform, false);
        _available.Add(obj);
    }

    private PooledObject CreateObject()
    {
        var go = Instantiate(_prefab, transform);
        go.name = _prefab.name;
        go.SetActive(false);

        var po = go.GetComponent<PooledObject>() ?? go.AddComponent<PooledObject>();
        po.Pool = this;
        return po;
    }
}
```

### PooledObject (자기 반환)
```csharp
public class PooledObject : MonoBehaviour
{
    public ObjectPool Pool { get; set; }

    public void ReturnToPool()
    {
        if (Pool != null)
            Pool.Return(this);
        else
            Destroy(gameObject);
    }

    public void ReturnAfter(float delay)
    {
        Invoke(nameof(ReturnToPool), delay);
    }
}
```

### 사용
```csharp
// 인스펙터에서 Pool 참조
[SerializeField] private ObjectPool _bulletPool;

void Fire()
{
    var bullet = _bulletPool.Get();
    bullet.transform.position = firePoint.position;
    bullet.GetComponent<Bullet>().Init(direction);
}

// Bullet.cs
void OnHit()
{
    GetComponent<PooledObject>().ReturnToPool();
}
```

---

## Advanced: Async FSM

UniTask 기반 비동기 상태 머신. 복잡한 게임 플로우에 적합.

### IState 인터페이스
```csharp
using Cysharp.Threading.Tasks;
using System.Threading;

public interface IState<TFSM>
{
    string Name { get; }
    UniTask Enter(CancellationToken token);
    UniTask Update(CancellationToken token);
    UniTask Leave(CancellationToken token);
    void AddTransition(string trigger, IState<TFSM> state);
    IState<TFSM> GetTransition(string trigger);
}
```

### FSM 베이스
```csharp
public abstract class FSM<TFSM> : MonoBehaviour where TFSM : FSM<TFSM>
{
    public IState<TFSM> CurrentState { get; private set; }
    private IState<TFSM> _nextState;
    private CancellationTokenSource _cts;

    protected virtual void OnDestroy()
    {
        _cts?.Cancel();
        _cts?.Dispose();
    }

    public void InitState(IState<TFSM> state)
    {
        _cts?.Cancel();
        _cts = new CancellationTokenSource();
        _nextState = state;
        UpdateLoopAsync(_cts.Token).Forget();
    }

    private async UniTaskVoid UpdateLoopAsync(CancellationToken token)
    {
        while (!token.IsCancellationRequested)
        {
            await ProcessState(token);
            await UniTask.NextFrame(token);
        }
    }

    private async UniTask ProcessState(CancellationToken token)
    {
        if (CurrentState != null)
            await CurrentState.Update(token);

        if (_nextState != null)
        {
            if (CurrentState != null)
                await CurrentState.Leave(token);

            CurrentState = _nextState;
            _nextState = null;

            if (CurrentState != null)
                await CurrentState.Enter(token);
        }
    }

    public void Trigger(string trigger)
    {
        if (CurrentState == null || _nextState != null) return;

        var next = CurrentState.GetTransition(trigger);
        if (next != null) _nextState = next;
    }
}
```

### State 베이스
```csharp
public abstract class State<TFSM> : IState<TFSM> where TFSM : FSM<TFSM>
{
    protected TFSM FSM { get; }
    public string Name { get; }

    private Dictionary<string, IState<TFSM>> _transitions = new();

    protected State(string name, TFSM fsm)
    {
        Name = name;
        FSM = fsm;
    }

    public void AddTransition(string trigger, IState<TFSM> state) => _transitions[trigger] = state;
    public IState<TFSM> GetTransition(string trigger) => _transitions.GetValueOrDefault(trigger);

    public async UniTask Enter(CancellationToken token) => await OnEnter(token);
    public async UniTask Update(CancellationToken token) => await OnUpdate(token);
    public async UniTask Leave(CancellationToken token) => await OnLeave(token);

    protected abstract UniTask OnEnter(CancellationToken token);
    protected abstract UniTask OnUpdate(CancellationToken token);
    protected abstract UniTask OnLeave(CancellationToken token);
}
```

### 사용 예
```csharp
public class GameFSM : FSM<GameFSM>
{
    void Start()
    {
        var idle = new IdleState("Idle", this);
        var play = new PlayState("Play", this);
        var result = new ResultState("Result", this);

        idle.AddTransition("start", play);
        play.AddTransition("win", result);
        play.AddTransition("lose", result);
        result.AddTransition("retry", idle);

        InitState(idle);
    }
}

public class PlayState : State<GameFSM>
{
    public PlayState(string name, GameFSM fsm) : base(name, fsm) { }

    protected override async UniTask OnEnter(CancellationToken token)
    {
        await UISystem.ShowAsync<UIPlay>(token);
    }

    protected override async UniTask OnUpdate(CancellationToken token)
    {
        // 게임 로직
        await UniTask.Yield();
    }

    protected override async UniTask OnLeave(CancellationToken token)
    {
        await UISystem.HideAsync<UIPlay>(token);
    }
}
```
