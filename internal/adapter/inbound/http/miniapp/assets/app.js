"use strict";

const statusLine = document.querySelector("#status");
const locateButton = document.querySelector("#locate");
const locationLine = document.querySelector("#location");

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
async function initialize() {
  const configResponse = await fetch("/miniapp/config.json"); if (!configResponse.ok) throw new Error("configuration unavailable");
  const config = await configResponse.json(); await liff.init({liffId: config.liffId});
  if (!liff.isLoggedIn()) { liff.login({redirectUri: window.location.origin + "/miniapp/"}); return; }
  const idToken = liff.getIDToken(); if (!idToken) throw new Error("LINE MINI App authentication is required");
  const response = await fetch("/miniapp/api/v1/session", {headers: {Authorization: `Bearer ${idToken}`}}); if (!response.ok) throw new Error("LINE session verification failed");
  locateButton.disabled = false; status("LINE 身分驗證成功。");
}
locateButton.addEventListener("click", async () => {
  locateButton.disabled = true; status("正在取得位置…");
  try { const reading = await currentLocation(); locationLine.textContent = `緯度 ${reading.latitude.toFixed(6)}，經度 ${reading.longitude.toFixed(6)}，精度約 ${Math.round(reading.accuracyM)} 公尺`; status("位置讀值有效；尚未送往任何業務功能。"); }
  catch (error) { status(({unsupported: "此環境不支援安全定位。", permission_denied: "未取得定位權限。", position_unavailable: "無法取得目前位置。", timeout: "取得位置逾時。", invalid_reading: "裝置回傳的位置無效。"})[error.code] || "取得位置失敗。"); }
  finally { locateButton.disabled = false; }
});
initialize().catch(error => status(error.message));
