"use strict";

const enrollmentForm = document.querySelector("#enrollment");
const enrollmentCode = document.querySelector("#enrollment-code");
const punchPanel = document.querySelector("#punch-panel");
const attendanceOption = document.querySelector("#attendance-option");
const clockIn = document.querySelector("#clock-in");
const clockOut = document.querySelector("#clock-out");
const statusLine = document.querySelector("#status");
let idToken = "";
let assignedOptions = [];

function showStatus(message) { statusLine.textContent = message; }

function errorCode(error) {
  if (error && error.code === 1) return "permission_denied";
  if (error && error.code === 2) return "position_unavailable";
  if (error && error.code === 3) return "timeout";
  return error && error.message ? error.message : "unknown_error";
}

async function request(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: {
      "Authorization": `Bearer ${idToken}`,
      ...(options.body ? { "Content-Type": "application/json" } : {})
    }
  });
  const payload = await response.json();
  if (!response.ok) throw new Error(payload.error?.code || "request_failed");
  return payload;
}

function renderOptions(payload) {
  enrollmentForm.hidden = !payload.enrollmentRequired;
  assignedOptions = payload.options || [];
  attendanceOption.replaceChildren();
  for (const option of assignedOptions) {
    const element = document.createElement("option");
    element.value = option.shiftInstanceId;
    element.textContent = `${option.label} — ${option.workplaceLabel}`;
    attendanceOption.append(element);
  }
  punchPanel.hidden = payload.enrollmentRequired || assignedOptions.length === 0;
  updateActions();
  if (!payload.enrollmentRequired && assignedOptions.length === 0) showStatus("目前沒有可打卡的指派班次。");
}

function updateActions() {
  const selected = assignedOptions.find((option) => option.shiftInstanceId === attendanceOption.value);
  clockIn.hidden = !selected || !selected.allowedKinds.includes("clock_in");
  clockOut.hidden = !selected || !selected.allowedKinds.includes("clock_out");
}

attendanceOption.addEventListener("change", updateActions);

async function loadOptions() {
  renderOptions(await request("/miniapp/api/v1/attendance/options"));
}

async function initialize() {
  const response = await fetch("/miniapp/config.json");
  if (!response.ok) throw new Error("config_unavailable");
  const config = await response.json();
  await liff.init({ liffId: config.liffId });
  if (!liff.isLoggedIn()) {
    liff.login({ redirectUri: window.location.origin + "/miniapp/" });
    return;
  }
  idToken = liff.getIDToken();
  if (!idToken) throw new Error("line_login_required");
  await loadOptions();
}

enrollmentForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  showStatus("正在登錄…");
  try {
    const result = await request("/miniapp/api/v1/attendance/enrollments", {
      method: "POST",
      body: JSON.stringify({ enrollmentCode: enrollmentCode.value })
    });
    showStatus(result.status === "already_enrolled" ? "已完成登錄。" : "登錄成功。");
    enrollmentCode.value = "";
    await loadOptions();
  } catch (error) {
    showStatus(`無法登錄：${errorCode(error)}`);
  }
});

async function punch(kind) {
  if (!navigator.geolocation) throw new Error("geolocation_unavailable");
  const selected = assignedOptions.find((option) => option.shiftInstanceId === attendanceOption.value);
  if (!selected || !selected.allowedKinds.includes(kind)) throw new Error("punch_not_available");
  const position = await new Promise((resolve, reject) => {
    navigator.geolocation.getCurrentPosition(resolve, reject, {
      enableHighAccuracy: false,
      maximumAge: 0,
      timeout: 10000
    });
  });
  return request("/miniapp/api/v1/attendance/punches", {
    method: "POST",
    body: JSON.stringify({
      shiftInstanceId: selected.shiftInstanceId,
      kind,
      latitude: position.coords.latitude,
      longitude: position.coords.longitude,
      accuracyMeters: position.coords.accuracy
    })
  });
}

for (const [button, kind] of [[clockIn, "clock_in"], [clockOut, "clock_out"]]) {
  button.addEventListener("click", async () => {
    clockIn.disabled = true;
    clockOut.disabled = true;
    showStatus("正在打卡…");
    try {
      const result = await punch(kind);
      const review = result.reviewState === "clear" ? "已接受" : `待覆核：${(result.reasons || []).join("、")}`;
      showStatus(`${kind === "clock_in" ? "上班" : "下班"}打卡成功（${result.recordedAt}，${review}）`);
      await loadOptions();
    } catch (error) {
      showStatus(`無法打卡：${errorCode(error)}`);
    } finally {
      clockIn.disabled = false;
      clockOut.disabled = false;
      updateActions();
    }
  });
}

initialize().catch((error) => showStatus(`無法載入出勤資料：${errorCode(error)}`));
