# Event Patterns

이벤트 통신 패턴: GameEvents (Static Events)와 GameEventManager (Type-based Events).

## Table of Contents
1. [GameEvents (C# Static Events)](#gameevents-c-static-events) - 코드 기반 이벤트 버스
2. [Advanced: GameEventManager](#advanced-gameeventmanager) - 타입 기반 이벤트

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
