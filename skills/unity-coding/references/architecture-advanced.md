# Advanced Patterns

프로젝트 성장 시 사용하는 고급 패턴: MonoBase, Component Pool, Async FSM, Disposable System, EventNotifier.

## Table of Contents
1. [MonoBase](#monobase) - IDisposable 자동 관리
2. [Component Pool](#component-pool) - 컴포넌트 기반 풀링
3. [Async FSM](#async-fsm) - UniTask 기반 상태 머신
4. [Disposable System](#disposable-system) - 구독 자동 정리
5. [EventNotifier](#eventnotifier) - Update 구독 패턴

---

## MonoBase

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

## Component Pool

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

## Async FSM

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

## Disposable System

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

## EventNotifier

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
