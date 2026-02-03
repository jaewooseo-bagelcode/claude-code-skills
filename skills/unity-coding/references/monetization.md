# Monetization

광고(IAA), 인앱 구매(IAP), GDPR/ATT 동의 처리.

## Table of Contents
1. [수익화 전략](#수익화-전략)
2. [AdsManager](#adsmanager) - 광고 통합
3. [IAPManager](#iapmanager) - 인앱 구매
4. [ConsentManager](#consentmanager) - GDPR/ATT

---

## 수익화 전략

### 하이퍼캐주얼 수익 구조
```
Revenue Split:
- Interstitial: 40%
- Rewarded: 50%
- Remove Ads IAP: 10%
```

### 광고 배치 규칙
| 광고 타입 | 타이밍 | 쿨다운 |
|----------|--------|--------|
| Interstitial | 레벨 완료 후 | 60초 |
| Rewarded | 유저 요청 (부활, 2x 보상) | 없음 |

**금지**: 실패 직후 Interstitial 노출

### IAP 상품 구성
| 타입 | 상품 | 가격대 |
|------|------|--------|
| Non-Consumable | Remove Ads | $2.99-4.99 |
| Non-Consumable | Premium Unlock | $4.99-9.99 |
| Consumable | 코인 팩 | $0.99-9.99 |

---

## AdsManager

Interstitial, Rewarded 광고 통합 관리.

```csharp
using UnityEngine;
using System;

public class AdsManager : MonoBehaviour
{
    public static AdsManager Instance { get; private set; }

    [Header("Settings")]
    [SerializeField] private bool testMode = true;
    [SerializeField] private float interstitialCooldown = 60f;

    [Header("Ad Unit IDs")]
    [SerializeField] private string interstitialId = "Interstitial_Android";
    [SerializeField] private string rewardedId = "Rewarded_Android";

    private float lastInterstitialTime = -9999f;
    private Action onRewardedComplete;
    private bool isInitialized;

    // Remove Ads IAP 구매 여부
    public bool AdsEnabled => !PlayerPrefs.HasKey("AdsRemoved");

    void Awake()
    {
        if (Instance != null)
        {
            Destroy(gameObject);
            return;
        }
        Instance = this;
        DontDestroyOnLoad(gameObject);
    }

    void Start()
    {
        InitializeAds();
    }

    public void InitializeAds()
    {
        if (isInitialized) return;

        #if UNITY_IOS
        interstitialId = "Interstitial_iOS";
        rewardedId = "Rewarded_iOS";
        #endif

        // TODO: SDK 초기화
        // Unity Ads: Advertisement.Initialize(gameId, testMode, this);
        // AdMob: MobileAds.Initialize(status => { });

        isInitialized = true;
        Debug.Log("[AdsManager] Initialized");

        LoadInterstitial();
        LoadRewarded();
    }

    #region Interstitial

    public void LoadInterstitial()
    {
        if (!AdsEnabled) return;
        // TODO: 광고 로드
        Debug.Log("[AdsManager] Loading interstitial...");
    }

    public bool CanShowInterstitial()
    {
        if (!AdsEnabled) return false;
        if (Time.time - lastInterstitialTime < interstitialCooldown) return false;
        // TODO: 로드 여부 체크
        return true;
    }

    public void ShowInterstitial()
    {
        if (!CanShowInterstitial())
        {
            Debug.Log("[AdsManager] Interstitial not ready or cooldown active");
            return;
        }

        lastInterstitialTime = Time.time;
        // TODO: 광고 표시
        Debug.Log("[AdsManager] Showing interstitial");

        LoadInterstitial(); // 다음 광고 로드
    }

    /// <summary>
    /// 레벨 완료 등 적절한 시점에 호출
    /// </summary>
    public void TryShowInterstitial()
    {
        if (CanShowInterstitial())
            ShowInterstitial();
    }

    #endregion

    #region Rewarded

    public void LoadRewarded()
    {
        if (!AdsEnabled) return;
        // TODO: 광고 로드
        Debug.Log("[AdsManager] Loading rewarded...");
    }

    public bool IsRewardedReady()
    {
        // TODO: 로드 여부 체크
        return true;
    }

    /// <summary>
    /// 보상형 광고 표시. 완료 시 onComplete 콜백 호출.
    /// </summary>
    public void ShowRewarded(Action onComplete)
    {
        if (!IsRewardedReady())
        {
            Debug.Log("[AdsManager] Rewarded ad not ready");
            return;
        }

        onRewardedComplete = onComplete;
        // TODO: 광고 표시
        Debug.Log("[AdsManager] Showing rewarded");

        // 테스트 모드: 즉시 보상
        if (testMode)
            OnRewardedComplete();
    }

    private void OnRewardedComplete()
    {
        Debug.Log("[AdsManager] Rewarded complete - granting reward");
        onRewardedComplete?.Invoke();
        onRewardedComplete = null;
        LoadRewarded();
    }

    #endregion

    /// <summary>
    /// IAP 구매 후 호출
    /// </summary>
    public void RemoveAds()
    {
        PlayerPrefs.SetInt("AdsRemoved", 1);
        PlayerPrefs.Save();
        Debug.Log("[AdsManager] Ads removed");
    }
}
```

### 사용 예
```csharp
// 레벨 완료 시
void OnLevelComplete()
{
    SaveProgress();
    AdsManager.Instance.TryShowInterstitial();
    LoadNextLevel();
}

// 부활 버튼
void OnReviveButtonClicked()
{
    AdsManager.Instance.ShowRewarded(() => {
        player.Revive();
    });
}
```

---

## IAPManager

인앱 구매 관리. Unity IAP 패키지 필요 (`com.unity.purchasing`).

```csharp
using UnityEngine;
using System;

public class IAPManager : MonoBehaviour
{
    public static IAPManager Instance { get; private set; }

    // Product IDs - App Store Connect / Google Play Console과 일치해야 함
    public const string REMOVE_ADS = "com.yourcompany.yourgame.removeads";
    public const string COINS_100 = "com.yourcompany.yourgame.coins100";
    public const string COINS_500 = "com.yourcompany.yourgame.coins500";
    public const string PREMIUM = "com.yourcompany.yourgame.premium";

    public bool IsInitialized { get; private set; }

    public event Action<string> OnPurchaseComplete;
    public event Action<string, string> OnPurchaseFailed;

    void Awake()
    {
        if (Instance != null)
        {
            Destroy(gameObject);
            return;
        }
        Instance = this;
        DontDestroyOnLoad(gameObject);
    }

    void Start()
    {
        InitializePurchasing();
    }

    void InitializePurchasing()
    {
        /*
        // Unity IAP 초기화
        var builder = ConfigurationBuilder.Instance(StandardPurchasingModule.Instance());

        // Non-consumables
        builder.AddProduct(REMOVE_ADS, ProductType.NonConsumable);
        builder.AddProduct(PREMIUM, ProductType.NonConsumable);

        // Consumables
        builder.AddProduct(COINS_100, ProductType.Consumable);
        builder.AddProduct(COINS_500, ProductType.Consumable);

        UnityPurchasing.Initialize(this, builder);
        */

        IsInitialized = true;
        Debug.Log("[IAPManager] Initialized");
    }

    public void BuyProduct(string productId)
    {
        if (!IsInitialized)
        {
            Debug.LogError("[IAPManager] Not initialized");
            OnPurchaseFailed?.Invoke(productId, "Not initialized");
            return;
        }

        Debug.Log($"[IAPManager] Purchasing: {productId}");

        // TODO: 실제 구매 처리
        // storeController.InitiatePurchase(productId);

        #if UNITY_EDITOR
        ProcessPurchase(productId); // 에디터 테스트
        #endif
    }

    private void ProcessPurchase(string productId)
    {
        Debug.Log($"[IAPManager] Processing purchase: {productId}");

        switch (productId)
        {
            case REMOVE_ADS:
                PlayerPrefs.SetInt("AdsRemoved", 1);
                AdsManager.Instance?.RemoveAds();
                break;

            case COINS_100:
                AddCoins(100);
                break;

            case COINS_500:
                AddCoins(500);
                break;

            case PREMIUM:
                PlayerPrefs.SetInt("Premium", 1);
                break;
        }

        PlayerPrefs.Save();
        OnPurchaseComplete?.Invoke(productId);
    }

    private void AddCoins(int amount)
    {
        int current = PlayerPrefs.GetInt("Coins", 0);
        PlayerPrefs.SetInt("Coins", current + amount);
        Debug.Log($"[IAPManager] Added {amount} coins. Total: {current + amount}");
    }

    public string GetLocalizedPrice(string productId)
    {
        // TODO: 스토어에서 실제 가격 가져오기
        // return storeController.products.WithID(productId)?.metadata.localizedPriceString;

        return productId switch
        {
            REMOVE_ADS => "$2.99",
            COINS_100 => "$0.99",
            COINS_500 => "$4.99",
            PREMIUM => "$4.99",
            _ => ""
        };
    }

    public void RestorePurchases()
    {
        Debug.Log("[IAPManager] Restoring purchases...");

        // TODO: 복원 구현
        // iOS: extensionProvider.GetExtension<IAppleExtensions>().RestoreTransactions(OnRestore);
        // Android: 자동 복원됨

        #if UNITY_IOS
        Debug.Log("[IAPManager] iOS restore initiated");
        #endif
    }

    public bool HasPurchased(string productId)
    {
        return productId switch
        {
            REMOVE_ADS => PlayerPrefs.GetInt("AdsRemoved", 0) == 1,
            PREMIUM => PlayerPrefs.GetInt("Premium", 0) == 1,
            _ => false
        };
    }
}
```

### 사용 예
```csharp
// 상점 UI
void OnRemoveAdsClicked()
{
    IAPManager.Instance.BuyProduct(IAPManager.REMOVE_ADS);
}

void OnEnable()
{
    IAPManager.Instance.OnPurchaseComplete += OnPurchased;
}

void OnDisable()
{
    IAPManager.Instance.OnPurchaseComplete -= OnPurchased;
}

void OnPurchased(string productId)
{
    if (productId == IAPManager.REMOVE_ADS)
        HideAdsButton();
}
```

---

## ConsentManager

GDPR (EU) 및 ATT (iOS 14.5+) 동의 처리. **광고 초기화 전에 반드시 호출**.

```csharp
using UnityEngine;
using System;

public class ConsentManager : MonoBehaviour
{
    public static ConsentManager Instance { get; private set; }

    private const string CONSENT_KEY = "GDPRConsent";
    private const string ATT_KEY = "ATTStatus";

    public bool HasGDPRConsent => PlayerPrefs.GetInt(CONSENT_KEY, -1) == 1;
    public bool GDPRConsentAsked => PlayerPrefs.GetInt(CONSENT_KEY, -1) != -1;

    public event Action<bool> OnGDPRConsentResult;
    public event Action<bool> OnATTConsentResult;

    void Awake()
    {
        if (Instance != null)
        {
            Destroy(gameObject);
            return;
        }
        Instance = this;
        DontDestroyOnLoad(gameObject);
    }

    void Start()
    {
        CheckConsent();
    }

    public void CheckConsent()
    {
        // EU 유저: GDPR 동의 필요
        if (!GDPRConsentAsked && ShouldShowGDPR())
        {
            ShowGDPRDialog();
            return;
        }

        // iOS 14.5+: ATT 동의 필요
        #if UNITY_IOS
        CheckATT();
        #else
        InitializeAdsWithConsent();
        #endif
    }

    bool ShouldShowGDPR()
    {
        // TODO: 지역 감지 (IP 기반 또는 디바이스 로케일)
        // 안전하게: 모든 유저에게 표시
        return true;
    }

    public void ShowGDPRDialog()
    {
        Debug.Log("[Consent] Showing GDPR dialog");
        // TODO: 동의 UI 표시
        // ConsentUI.Show(OnGDPRResult);

        #if UNITY_EDITOR
        OnGDPRResult(true); // 에디터 테스트
        #endif
    }

    public void OnGDPRResult(bool accepted)
    {
        PlayerPrefs.SetInt(CONSENT_KEY, accepted ? 1 : 0);
        PlayerPrefs.Save();

        Debug.Log($"[Consent] GDPR consent: {(accepted ? "Accepted" : "Denied")}");
        OnGDPRConsentResult?.Invoke(accepted);

        #if UNITY_IOS
        CheckATT();
        #else
        InitializeAdsWithConsent();
        #endif
    }

    #if UNITY_IOS
    void CheckATT()
    {
        // 필요 패키지: com.unity.ads.ios-support
        /*
        var status = ATTrackingStatusBinding.GetAuthorizationTrackingStatus();

        if (status == ATTrackingStatusBinding.AuthorizationTrackingStatus.NOT_DETERMINED)
        {
            ATTrackingStatusBinding.RequestAuthorizationTracking();
        }
        else
        {
            OnATTResult(status == ATTrackingStatusBinding.AuthorizationTrackingStatus.AUTHORIZED);
        }
        */

        Debug.Log("[Consent] ATT check (iOS)");
        InitializeAdsWithConsent();
    }

    void OnATTResult(bool authorized)
    {
        PlayerPrefs.SetInt(ATT_KEY, authorized ? 1 : 0);
        PlayerPrefs.Save();

        Debug.Log($"[Consent] ATT: {(authorized ? "Authorized" : "Denied")}");
        OnATTConsentResult?.Invoke(authorized);

        InitializeAdsWithConsent();
    }
    #endif

    void InitializeAdsWithConsent()
    {
        bool personalizedAds = HasGDPRConsent;

        #if UNITY_IOS
        personalizedAds = personalizedAds && PlayerPrefs.GetInt(ATT_KEY, 0) == 1;
        #endif

        Debug.Log($"[Consent] Initializing ads with personalization: {personalizedAds}");

        // 광고 SDK에 동의 상태 전달
        // Unity Ads: MetaData("gdpr").Set("consent", personalizedAds ? "true" : "false");
        // AdMob: RequestConfiguration with TagForUnderAgeOfConsent

        AdsManager.Instance?.InitializeAds();
    }

    /// <summary>
    /// 설정 메뉴에서 동의 초기화
    /// </summary>
    public void ResetConsent()
    {
        PlayerPrefs.DeleteKey(CONSENT_KEY);
        PlayerPrefs.DeleteKey(ATT_KEY);
        PlayerPrefs.Save();
        Debug.Log("[Consent] Consent reset");
    }
}
```

### 초기화 순서
```
1. ConsentManager.CheckConsent()
   ├─ GDPR 동의 안 받음 → ShowGDPRDialog()
   ├─ iOS: ATT 동의 요청
   └─ InitializeAdsWithConsent()
       └─ AdsManager.InitializeAds()
```

### 주의사항
- **광고 SDK 초기화 전**에 동의 확인 필수
- iOS: ATT 프롬프트는 **앱 시작 직후** 표시 권장
- GDPR: EU 유저에게만 표시해도 되지만, 전체 표시가 안전
