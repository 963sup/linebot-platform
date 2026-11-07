"use strict";

const enrollmentForm = document.querySelector("#enrollment");
const enrollmentCode = document.querySelector("#enrollment-code");
const attendancePanel = document.querySelector("#attendance-panel");
const workdayLine = document.querySelector("#workday");
const clockInLine = document.querySelector("#clock-in-time");
const clockOutLine = document.querySelector("#clock-out-time");
const clockIn = document.querySelector("#clock-in");
const clockOut = document.querySelector("#clock-out");
const statusLine = document.querySelector("#status");
let idToken = "";
let today = null;

function showStatus(message) { statusLine.textContent = message; }

function errorCode(error) {
  if (error && error.code === 1) return "permission_denied";
  if (error && error.code === 2) return "position_unavailable";
  if (error && error.code === 3) return "timeout";
  return error && error.code ? error.code : "unknown_error";
}

function errorMessage(error) {
  const messages = {
    unauthorized: "LINE 登入已失效，請重新開啟。",
    enrollment_required: "請先完成員工登錄。",
    permission_denied: "未取得定位權限。",
    position_unavailable: "無法取得目前位置。",
    timeout: "取得位置逾時，請重試。",
    attendance_conflict: "今日打卡狀態已更新。",
    attendance_unavailable: "出勤服務暫時無法使用。"
  };
  return messages[errorCode(error)] || "操作失敗，請稍後重試。";
}

async function request(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: {
      "Authorization": `Bearer ${idToken}`,
      ...(options.body ? { "Content-Type": "application/json" } : {})
    }
  });
  const contentType = response.headers.get("content-type") || "";
  const payload = contentType.includes("application/json") ? await response.json() : null;
  if (!response.ok) {
    const error = new Error("request_failed");
    error.code = payload?.error?.code || `http_${response.status}`;
    throw error;
  }
  if (!payload) {
    const error = new Error("invalid_response");
    error.code = "invalid_response";
    throw error;
  }
  return payload;
}

function formatTime(value) {
  return value ? new Date(value).toLocaleTimeString("zh-TW", {hour: "2-digit", minute: "2-digit"}) : "尚未打卡";
}

function renderToday(payload) {
  today = payload;
  enrollmentForm.hidden = !payload.enrollmentRequired;
  attendancePanel.hidden = payload.enrollmentRequired;
  if (payload.enrollmentRequired) return;
  workdayLine.textContent = `日期：${payload.workday}`;
  clockInLine.textContent = `上班：${formatTime(payload.clockIn?.recordedAt)}`;
  clockOutLine.textContent = `下班：${formatTime(payload.clockOut?.recordedAt)}`;
  const allowed = payload.allowedKinds || [];
  clockIn.hidden = !allowed.includes("clock_in");
  clockOut.hidden = !allowed.includes("clock_out");
  if (allowed.length === 0) showStatus("今日打卡已完成。");
}

async function loadToday() { renderToday(await request("/miniapp/api/v1/attendance/today")); }

async function initialize() {
  const response = await fetch("/miniapp/config.json");
  if (!response.ok) throw Object.assign(new Error("config_unavailable"), {code: "config_unavailable"});
  const config = await response.json();
  await liff.init({ liffId: config.liffId });
  if (!liff.isLoggedIn()) {
    liff.login({ redirectUri: window.location.origin + "/miniapp/" });
    return;
  }
  idToken = liff.getIDToken();
  if (!idToken) throw Object.assign(new Error("line_login_required"), {code: "unauthorized"});
  await loadToday();
}

enrollmentForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  showStatus("正在登錄…");
  try {
    const result = await request("/miniapp/api/v1/attendance/enrollments", {
      method: "POST", body: JSON.stringify({ enrollmentCode: enrollmentCode.value })
    });
    showStatus(result.status === "already_enrolled" ? "已完成登錄。" : "登錄成功。");
    enrollmentCode.value = "";
    await loadToday();
  } catch (error) { showStatus(errorMessage(error)); }
});

async function punch(kind) {
  if (!navigator.geolocation) throw Object.assign(new Error("geolocation_unavailable"), {code: "position_unavailable"});
  if (!today || !(today.allowedKinds || []).includes(kind)) throw Object.assign(new Error("punch_not_available"), {code: "attendance_conflict"});
  const position = await new Promise((resolve, reject) => {
    navigator.geolocation.getCurrentPosition(resolve, reject, {enableHighAccuracy: false, maximumAge: 0, timeout: 10000});
  });
  return request("/miniapp/api/v1/attendance/punches", {
    method: "POST",
    body: JSON.stringify({kind, latitude: position.coords.latitude, longitude: position.coords.longitude, accuracyMeters: position.coords.accuracy})
  });
}

for (const [button, kind] of [[clockIn, "clock_in"], [clockOut, "clock_out"]]) {
  button.addEventListener("click", async () => {
    clockIn.disabled = true;
    clockOut.disabled = true;
    showStatus("正在打卡…");
    try {
      const result = await punch(kind);
      showStatus(`${kind === "clock_in" ? "上班" : "下班"}打卡成功（${formatTime(result.recordedAt)}）`);
    } catch (error) { showStatus(errorMessage(error)); }
    finally {
      try { await loadToday(); } catch (_) { /* retain the last safe status */ }
      clockIn.disabled = false;
      clockOut.disabled = false;
    }
  });
}

initialize().catch((error) => showStatus(errorMessage(error)));
