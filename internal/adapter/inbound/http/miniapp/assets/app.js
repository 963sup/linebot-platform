"use strict";

const statusLine = document.querySelector("#status");
const locateButton = document.querySelector("#locate");
const locationLine = document.querySelector("#location");
const loginButton = document.querySelector("#login");

function status(message) { statusLine.textContent = message; }
export function validateLocationReading(coords, capturedAt = new Date().toISOString()) {
  const latitude = Number(coords?.latitude), longitude = Number(coords?.longitude), accuracyM = Number(coords?.accuracy);
  if (!Number.isFinite(latitude) || latitude < -90 || latitude > 90 || !Number.isFinite(longitude) || longitude < -180 || longitude > 180 || !Number.isFinite(accuracyM) || accuracyM < 0) throw Object.assign(new Error("invalid location reading"), {code: "invalid_reading"});
  return {latitude, longitude, accuracyM, capturedAt};
}
export function currentLocation(options = {}) {
  if (!window.isSecureContext || !navigator.geolocation) return Promise.reject(Object.assign(new Error("geolocation unsupported"), {code: "unsupported"}));
  return new Promise((resolve, reject) => navigator.geolocation.getCurrentPosition(
    position => { try { resolve(validateLocationReading(position.coords, new Date(position.timestamp).toISOString())); } catch (error) { reject(error); } },
    error => reject(Object.assign(new Error("geolocation failed"), {code: ({1: "permission_denied", 2: "position_unavailable", 3: "timeout"})[error.code] || "unknown"})),
    {enableHighAccuracy: options.enableHighAccuracy ?? true, timeout: options.timeout ?? 10000, maximumAge: options.maximumAge ?? 0}
  ));
}
async function initialize(requestConsent = false) {
  locateButton.disabled = true;
  loginButton.hidden = true;
  const configResponse = await fetch("/miniapp/config.json"); if (!configResponse.ok) throw new Error("configuration unavailable");
  const config = await configResponse.json(); await liff.init({liffId: config.liffId});
  if (!liff.isLoggedIn()) { liff.login({redirectUri: window.location.origin + "/miniapp/"}); return; }
  const permission = await liff.permission.query("openid");
  if (permission.state === "unavailable") throw new Error("此 MINI App 未啟用 openid，需由管理者修正 LINE Channel 設定；重新登入無法解決。");
  if (permission.state === "prompt") {
    loginButton.hidden = false;
    loginButton.textContent = "同意 LINE 身分授權";
    if (!requestConsent) { status("需要您的 LINE 身分授權，完成後即可使用定位。"); return; }
    try { await liff.permission.requestAll(); }
    catch { throw new Error("授權未完成。請重試；若持續失敗，管理者需確認 Channel consent simplification 已啟用。"); }
  }
  const idToken = liff.getIDToken();
  if (!idToken) {
    loginButton.hidden = false;
    throw new Error("尚未取得 LINE 身分憑證，無法啟用定位。請重新登入並同意授權；若仍失敗，請管理者確認此 MINI App 的 openid scope 與 LIFF ID 設定。");
  }
  const response = await fetch("/miniapp/api/v1/session", {headers: {Authorization: `Bearer ${idToken}`}});
  if (!response.ok) throw new Error(response.status === 401 ? "LINE 身分驗證失敗；請重新驗證，若持續失敗需檢查 Channel ID。" : "驗證服務暫時無法使用，請重試。");
  const session = await response.json();
  if (session.authenticated !== true) throw new Error("驗證服務未確認登入身分，請重試。");
  loginButton.hidden = true;
  locateButton.disabled = false; status("LINE 身分驗證成功。");
}
async function authenticate(requestConsent = false) {
  loginButton.disabled = true;
  loginButton.textContent = "重新驗證 LINE 身分";
  try { await initialize(requestConsent); }
  catch (error) {
    locateButton.disabled = true;
    loginButton.hidden = false;
    status(error.message || "驗證未完成，請重試。");
  } finally { loginButton.disabled = false; }
}
loginButton.addEventListener("click", () => authenticate(true));
locateButton.addEventListener("click", async () => {
  locateButton.disabled = true; status("正在取得位置…");
  try { const reading = await currentLocation(); locationLine.textContent = `緯度 ${reading.latitude.toFixed(6)}，經度 ${reading.longitude.toFixed(6)}，精度約 ${Math.round(reading.accuracyM)} 公尺`; status("位置讀值有效；尚未送往任何業務功能。"); }
  catch (error) { status(({unsupported: "此環境不支援安全定位。", permission_denied: "未取得定位權限。", position_unavailable: "無法取得目前位置。", timeout: "取得位置逾時。", invalid_reading: "裝置回傳的位置無效。"})[error.code] || "取得位置失敗。"); }
  finally { locateButton.disabled = false; }
});
authenticate();
