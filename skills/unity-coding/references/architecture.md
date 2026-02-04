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
9. [Advanced: Disposable System](#advanced-disposable-system) - 구독 자동 정리
10. [Advanced: EventNotifier](#advanced-eventnotifier) - Update 구독 패턴
11. [Advanced: GameEventManager](#advanced-gameeventmanager) - 타입 기반 이벤트

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

---

## Advanced: Disposable System

IDisposable 기반 구독 관리. 이벤트 누수 방지 및 자동 정리.

### DisposableAction
```csharp
using System;

public class DisposableAction : IDisposable
{
    private Action _onDispose;

    public static IDisposable Create(Action onDispose) => new DisposableAction(onDispose);

    private DisposableAction(Action onDispose) => _onDispose = onDispose;

    public void Dispose()
    {
        _onDispose?.Invoke();
        _onDispose = null;
    }
}
```

### CompositeDisposable
```csharp
using System;
using System.Collections.Generic;

public class CompositeDisposable : IDisposable
{
    private readonly List<IDisposable> _disposables = new();

    public void Add(IDisposable disposable) => _disposables.Add(disposable);

    public void Clear()
    {
        foreach (var d in _disposables)
            d?.Dispose();
        _disposables.Clear();
    }

    public void Dispose() => Clear();
}

public static class DisposableExtensions
{
    public static void AddTo(this IDisposable disposable, CompositeDisposable composite)
    {
        composite.Add(disposable);
    }
}
```

### 사용 예
```csharp
public class EnemyController : MonoBehaviour
{
    private CompositeDisposable _disposables = new();

    void OnEnable()
    {
        // 이벤트 구독을 IDisposable로 래핑
        GameEvents.OnGamePaused += OnPause;
        DisposableAction.Create(() => GameEvents.OnGamePaused -= OnPause)
            .AddTo(_disposables);
    }

    void OnDisable()
    {
        _disposables.Clear();  // 모든 구독 자동 해제
    }
}
```

---

## Advanced: EventNotifier

IDisposable 반환하는 이벤트 발행/구독 시스템. Update 중앙화에 적합.

### IEventSubscriber 인터페이스
```csharp
using System;

public interface IEventSubscriber
{
    IDisposable Subscribe(Action action);
}

public interface IEventSubscriber<T>
{
    IDisposable Subscribe(Action<T> action);
}
```

### EventNotifier 구현
```csharp
using System;
using System.Collections.Generic;
using UnityEngine;

public class EventNotifier : IEventSubscriber, IDisposable
{
    private readonly List<Action> _actions = new();
    private readonly List<Action> _temp = new();

    public IDisposable Subscribe(Action action)
    {
        if (action == null) throw new ArgumentNullException(nameof(action));
        _actions.Add(action);
        return DisposableAction.Create(() => _actions.Remove(action));
    }

    public void Dispatch()
    {
        _temp.Clear();
        _temp.AddRange(_actions);
        foreach (var action in _temp)
        {
            try { action?.Invoke(); }
            catch (Exception e) { Debug.LogException(e); }
        }
    }

    public void Dispose() => _actions.Clear();
}

public class EventNotifier<T> : IEventSubscriber<T>, IDisposable
{
    private readonly List<Action<T>> _actions = new();
    private readonly List<Action<T>> _temp = new();

    public IDisposable Subscribe(Action<T> action)
    {
        if (action == null) throw new ArgumentNullException(nameof(action));
        _actions.Add(action);
        return DisposableAction.Create(() => _actions.Remove(action));
    }

    public void Dispatch(T value)
    {
        _temp.Clear();
        _temp.AddRange(_actions);
        foreach (var action in _temp)
        {
            try { action?.Invoke(value); }
            catch (Exception e) { Debug.LogException(e); }
        }
    }

    public void Dispose() => _actions.Clear();
}
```

### Update 구독 패턴 (GameSystem 확장)
```csharp
public class GameSystem : Singleton<GameSystem>
{
    // Update 구독자
    private readonly EventNotifier _onUpdate = new();
    private readonly EventNotifier _onLateUpdate = new();
    private readonly EventNotifier<bool> _onApplicationPause = new();

    public static IEventSubscriber SubUpdate => Instance._onUpdate;
    public static IEventSubscriber SubLateUpdate => Instance._onLateUpdate;
    public static IEventSubscriber<bool> SubOnPause => Instance._onApplicationPause;

    void Update() => _onUpdate.Dispatch();
    void LateUpdate() => _onLateUpdate.Dispatch();
    void OnApplicationPause(bool pause) => _onApplicationPause.Dispatch(pause);

    protected override void OnDestroy()
    {
        base.OnDestroy();
        _onUpdate.Dispose();
        _onLateUpdate.Dispose();
        _onApplicationPause.Dispose();
    }
}
```

### Manager에서 Update 구독
```csharp
public class AdManager : IManager
{
    private IDisposable _updateSub;
    private CompositeDisposable _disposables = new();

    public void Init()
    {
        // Update 구독 (IDisposable 반환)
        _updateSub = GameSystem.SubUpdate.Subscribe(OnUpdate);

        // 또는 CompositeDisposable 사용
        GameSystem.SubLateUpdate.Subscribe(OnLateUpdate).AddTo(_disposables);
    }

    public void Clear()
    {
        _updateSub?.Dispose();
        _disposables.Clear();
    }

    private void OnUpdate() { /* 매 프레임 로직 */ }
    private void OnLateUpdate() { /* LateUpdate 로직 */ }
}
```

> **UpdateManager vs EventNotifier**: UpdateManager는 IUpdatable 인터페이스 기반, EventNotifier는 IDisposable 기반. EventNotifier가 구독 해제 실수를 방지하는 데 더 안전.

---

## Advanced: GameEventManager

타입 기반 이벤트 시스템. 이벤트를 클래스로 정의하여 데이터 전달 및 재사용.

### IGameEvent 인터페이스
```csharp
public interface IGameEvent { }
```

### GameEventDispatcher
```csharp
using System;
using System.Collections.Generic;
using UnityEngine;

public class GameEventDispatcher<T> where T : IGameEvent
{
    private readonly HashSet<Action<T>> _checker = new();
    private readonly List<Action<T>> _handlers = new();
    private readonly List<Action<T>> _temp = new();

    public void Subscribe(Action<T> handler)
    {
        if (handler == null) return;
        if (_checker.Add(handler))
            _handlers.Add(handler);
    }

    public void Unsubscribe(Action<T> handler)
    {
        if (handler == null) return;
        if (_checker.Remove(handler))
            _handlers.Remove(handler);
    }

    public void Dispatch(T evt)
    {
        _temp.Clear();
        _temp.AddRange(_handlers);
        foreach (var handler in _temp)
        {
            try { handler(evt); }
            catch (Exception e) { Debug.LogError($"[{typeof(T).Name}] {e}"); }
        }
    }

    public void Clear()
    {
        _handlers.Clear();
        _checker.Clear();
    }
}
```

### GameEventManager
```csharp
using System;
using System.Collections.Generic;

public class GameEventManager : IManager
{
    private readonly Dictionary<Type, object> _dispatchers = new();

    public bool IsInit { get; private set; }

    public void Init() => IsInit = true;

    public void Clear()
    {
        foreach (var dispatcher in _dispatchers.Values)
        {
            var method = dispatcher.GetType().GetMethod("Clear");
            method?.Invoke(dispatcher, null);
        }
        _dispatchers.Clear();
        IsInit = false;
    }

    public void Subscribe<T>(Action<T> handler) where T : IGameEvent
    {
        var dispatcher = GetOrCreateDispatcher<T>();
        dispatcher.Subscribe(handler);
    }

    public void Unsubscribe<T>(Action<T> handler) where T : IGameEvent
    {
        if (_dispatchers.TryGetValue(typeof(T), out var obj))
        {
            var dispatcher = (GameEventDispatcher<T>)obj;
            dispatcher.Unsubscribe(handler);
        }
    }

    public void Publish<T>(T evt) where T : IGameEvent
    {
        if (_dispatchers.TryGetValue(typeof(T), out var obj))
        {
            var dispatcher = (GameEventDispatcher<T>)obj;
            dispatcher.Dispatch(evt);
        }
    }

    private GameEventDispatcher<T> GetOrCreateDispatcher<T>() where T : IGameEvent
    {
        var type = typeof(T);
        if (!_dispatchers.TryGetValue(type, out var obj))
        {
            obj = new GameEventDispatcher<T>();
            _dispatchers[type] = obj;
        }
        return (GameEventDispatcher<T>)obj;
    }
}
```

### GameEventSystem (Static API)
```csharp
public static class GameEventSystem
{
    private static GameEventManager Manager => GameEventManager.Instance;

    public static void Subscribe<T>(Action<T> handler) where T : IGameEvent
        => Manager?.Subscribe(handler);

    public static void Unsubscribe<T>(Action<T> handler) where T : IGameEvent
        => Manager?.Unsubscribe(handler);

    public static void Publish<T>(T evt) where T : IGameEvent
        => Manager?.Publish(evt);
}
```

### 이벤트 정의 예
```csharp
// 이벤트 클래스 정의
public class PlayerDamagedEvent : IGameEvent
{
    public float Damage { get; set; }
    public float CurrentHealth { get; set; }
    public Vector3 HitPoint { get; set; }
}

public class ScoreChangedEvent : IGameEvent
{
    public int OldScore { get; set; }
    public int NewScore { get; set; }
    public int Delta => NewScore - OldScore;
}

public class LevelCompletedEvent : IGameEvent
{
    public int LevelIndex { get; set; }
    public float ClearTime { get; set; }
    public int Stars { get; set; }
}
```

### 사용 예
```csharp
// ===== 발행 (Publisher) =====
public class Player : MonoBehaviour
{
    void TakeDamage(float damage, Vector3 hitPoint)
    {
        currentHealth -= damage;

        GameEventSystem.Publish(new PlayerDamagedEvent
        {
            Damage = damage,
            CurrentHealth = currentHealth,
            HitPoint = hitPoint
        });

        if (currentHealth <= 0)
            Die();
    }
}

// ===== 구독 (Subscriber) =====
public class DamageUI : MonoBehaviour
{
    void OnEnable()
    {
        GameEventSystem.Subscribe<PlayerDamagedEvent>(OnPlayerDamaged);
    }

    void OnDisable()
    {
        GameEventSystem.Unsubscribe<PlayerDamagedEvent>(OnPlayerDamaged);
    }

    private void OnPlayerDamaged(PlayerDamagedEvent evt)
    {
        ShowDamageText(evt.Damage, evt.HitPoint);
        UpdateHealthBar(evt.CurrentHealth);
    }
}

// ===== 다중 이벤트 구독 =====
public class AnalyticsManager : MonoBehaviour
{
    void OnEnable()
    {
        GameEventSystem.Subscribe<PlayerDamagedEvent>(OnDamage);
        GameEventSystem.Subscribe<ScoreChangedEvent>(OnScore);
        GameEventSystem.Subscribe<LevelCompletedEvent>(OnLevelComplete);
    }

    void OnDisable()
    {
        GameEventSystem.Unsubscribe<PlayerDamagedEvent>(OnDamage);
        GameEventSystem.Unsubscribe<ScoreChangedEvent>(OnScore);
        GameEventSystem.Unsubscribe<LevelCompletedEvent>(OnLevelComplete);
    }

    private void OnDamage(PlayerDamagedEvent evt)
        => LogEvent("player_damaged", ("damage", evt.Damage));

    private void OnScore(ScoreChangedEvent evt)
        => LogEvent("score_changed", ("delta", evt.Delta));

    private void OnLevelComplete(LevelCompletedEvent evt)
        => LogEvent("level_complete", ("level", evt.LevelIndex), ("time", evt.ClearTime));
}
```

### GameEvents vs GameEventManager

| 항목 | GameEvents (Static) | GameEventManager (Type-based) |
|------|---------------------|-------------------------------|
| 이벤트 정의 | `event Action<T>` | `class : IGameEvent` |
| 데이터 전달 | 매개변수 제한 | 클래스로 무제한 |
| 타입 안전성 | 중간 | 높음 |
| 재사용성 | 낮음 | 높음 (이벤트 객체 재사용 가능) |
| 복잡도 | 낮음 | 중간 |
| 권장 규모 | 프로토타입/소규모 | 중규모 이상 |

> **선택 기준**: 프로토타입은 GameEvents로 빠르게, 규모가 커지면 GameEventManager로 전환.
