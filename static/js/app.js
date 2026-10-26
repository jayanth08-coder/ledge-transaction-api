(function () {
  var MAX_LOG = 40;
  var STORAGE_KEY = "ledger_json_panel_open";

  function isPanelOpen() {
    return localStorage.getItem(STORAGE_KEY) === "1";
  }

  function setPanelOpen(open) {
    localStorage.setItem(STORAGE_KEY, open ? "1" : "0");
    var panel = document.getElementById("json-status-panel");
    var fab = document.getElementById("json-panel-fab");
    if (panel) panel.classList.toggle("is-collapsed", !open);
    if (fab) fab.classList.toggle("is-visible", !open);
  }

  function tryParseJSON(text) {
    if (!text) return null;
    try {
      return JSON.parse(text);
    } catch (_) {
      return null;
    }
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function appendJsonLog(entry) {
    var log = document.getElementById("json-status-log");
    if (!log) return;
    var block = document.createElement("div");
    block.className = "json-log-entry";
    var statusClass = entry.status >= 400 ? "text-danger" : entry.status >= 300 ? "text-warning" : "text-success";
    var header = document.createElement("div");
    header.className = "json-log-meta";
    header.innerHTML =
      '<span class="badge text-bg-dark me-1">' +
      escapeHtml(entry.method || "?") +
      "</span>" +
      '<code class="me-2">' +
      escapeHtml(entry.path || "") +
      "</code>" +
      '<span class="' +
      statusClass +
      ' fw-semibold">HTTP ' +
      (entry.status || "?") +
      "</span>" +
      (entry.errorCode ? ' <span class="badge text-bg-warning ms-1">' + escapeHtml(entry.errorCode) + "</span>" : "");
    var pre = document.createElement("pre");
    pre.className = "json-log-body mb-0";
    pre.textContent = JSON.stringify(entry.body, null, 2);
    block.appendChild(header);
    block.appendChild(pre);
    log.appendChild(block);
    while (log.children.length > MAX_LOG) log.removeChild(log.firstChild);
    if (isPanelOpen()) log.scrollTop = log.scrollHeight;
  }

  function requestPath(detail) {
    var path =
      (detail.pathInfo && detail.pathInfo.requestPath) ||
      (detail.requestConfig && detail.requestConfig.path) ||
      "";
    if (!path && detail.elt && detail.elt.getAttribute) {
      path = detail.elt.getAttribute("hx-post") || detail.elt.getAttribute("hx-get") || "";
    }
    try {
      if (path.indexOf("http") === 0) path = new URL(path).pathname + new URL(path).search;
    } catch (_) {}
    return path;
  }

  function isApiPath(path) {
    return typeof path === "string" && path.indexOf("/api/") !== -1;
  }

  function formatMoney(rupees) {
    return "₹" + String(rupees);
  }

  function shortID(id) {
    if (!id) return "";
    return id.length <= 8 ? id : id.slice(0, 8) + "…";
  }

  function apiTxnURL(accountID, page, limit, filterType) {
    var url = "/api/accounts/" + encodeURIComponent(accountID) + "/transactions?page=" + page + "&limit=" + limit;
    if (filterType) url += "&type=" + encodeURIComponent(filterType);
    return url;
  }

  function successMessage(body) {
    var data = body && body.data;
    if (!data) return "Success.";
    if (data.reference_id && data.type) {
      return "Transaction recorded successfully (" + data.type + ") · ₹" + data.amount + " · Reference ID: " + data.reference_id;
    }
    if (data.reference_id && data.from_account_id) {
      return "Transfer completed successfully · ₹" + data.amount + " · Reference ID: " + data.reference_id;
    }
    if (data.name && data.id) {
      return "Account created successfully: " + data.name;
    }
    return "Success.";
  }

  function renderStatusAlert(target, status, body) {
    if (!target) return;
    var success = !!(body && body.success);
    var code = (body && body.error && body.error.code) || (success ? "OK" : "");
    var message = (body && body.error && body.error.message) || (success ? successMessage(body) : "Request failed.");

    var kindClass = "alert-danger";
    var badgeClass = "text-bg-danger";
    if (success || status === 201) {
      kindClass = "alert-success";
      badgeClass = "text-bg-success";
    } else if (status === 409 || code === "DUPLICATE_REFERENCE_ID" || code === "DUPLICATE_TRANSFER_REFERENCE") {
      kindClass = "alert-warning border border-warning";
      badgeClass = "text-bg-warning text-dark";
    }

    var extra = "";
    if (code === "DUPLICATE_REFERENCE_ID") {
      extra = '<div class="mb-1"><strong>Duplicate reference ID remarked:</strong> this reference is already used on this account.</div>';
    }

    target.innerHTML =
      '<div class="alert ' +
      kindClass +
      ' alert-dismissible fade show status-indication" role="alert" data-http-status="' +
      status +
      '" data-error-code="' +
      escapeHtml(code) +
      '"><div class="d-flex align-items-start gap-2"><span class="badge ' +
      badgeClass +
      ' fs-6">HTTP ' +
      status +
      '</span><div><div class="fw-semibold">' +
      escapeHtml(code || "ERROR") +
      "</div>" +
      extra +
      "<div>" +
      escapeHtml(message) +
      '</div></div></div><button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button></div>';
  }

  function isTxnListPayload(body) {
    return !!(
      body &&
      body.success &&
      body.data &&
      Array.isArray(body.data.data) &&
      body.data.pagination &&
      typeof body.data.pagination.current_page !== "undefined"
    );
  }

  function filterClass(current, target) {
    return current === target ? "btn btn-sm btn-primary" : "btn btn-sm btn-outline-primary";
  }

  // Re-render history from backend paginated JSON (page/limit applied server-side).
  function renderTxnPanelFromAPI(path, body) {
    var panel = document.getElementById("txn-panel");
    if (!panel || !isTxnListPayload(body)) return;

    var accountID = panel.getAttribute("data-account-id") || "";
    var m = path.match(/\/api\/accounts\/([^/]+)\/transactions/);
    if (m) accountID = decodeURIComponent(m[1]);

    var filterType = "";
    try {
      filterType = new URL(path, window.location.origin).searchParams.get("type") || "";
    } catch (_) {}

    var rows = body.data.data;
    var pag = body.data.pagination;
    var page = pag.current_page || 1;
    var limit = pag.page_size || 3;
    var totalPages = pag.total_pages || 0;
    var balance = panel.getAttribute("data-balance") || "0";

    panel.setAttribute("data-account-id", accountID);
    panel.setAttribute("data-filter", filterType);
    panel.setAttribute("data-page", String(page));
    panel.setAttribute("data-limit", String(limit));
    panel.setAttribute("hx-get", apiTxnURL(accountID, page, limit, filterType));

    var balEl = document.getElementById("account-balance");
    if (balEl) balEl.textContent = formatMoney(balance);

    var filters = document.getElementById("txn-filters");
    if (filters) {
      filters.innerHTML =
        '<button type="button" class="' +
        filterClass(filterType, "") +
        '" hx-get="' +
        apiTxnURL(accountID, 1, limit, "") +
        '" hx-target="#txn-panel" hx-swap="none" hx-headers=\'{"Accept":"application/json"}\'>All</button>' +
        '<button type="button" class="' +
        filterClass(filterType, "credit") +
        '" hx-get="' +
        apiTxnURL(accountID, 1, limit, "credit") +
        '" hx-target="#txn-panel" hx-swap="none" hx-headers=\'{"Accept":"application/json"}\'>Credits</button>' +
        '<button type="button" class="' +
        filterClass(filterType, "debit") +
        '" hx-get="' +
        apiTxnURL(accountID, 1, limit, "debit") +
        '" hx-target="#txn-panel" hx-swap="none" hx-headers=\'{"Accept":"application/json"}\'>Debits</button>';
      if (window.htmx) htmx.process(filters);
    }

    var tbody = document.getElementById("txn-table-body");
    if (tbody) {
      if (!rows.length) {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted py-4">No transactions found.</td></tr>';
      } else {
        tbody.innerHTML = rows
          .map(function (t) {
            var badge =
              t.type === "credit"
                ? '<span class="badge text-bg-success">credit</span>'
                : '<span class="badge text-bg-danger">debit</span>';
            var created = t.created_at ? String(t.created_at).replace("T", " ").replace("Z", "").slice(0, 19) : "";
            return (
              "<tr><td class=\"text-nowrap\">" +
              escapeHtml(created) +
              "</td><td>" +
              badge +
              '</td><td class="font-monospace">' +
              escapeHtml(formatMoney(t.amount)) +
              '</td><td class="font-monospace small">' +
              escapeHtml(t.reference_id || "") +
              "</td><td>" +
              escapeHtml(t.description || "") +
              '</td><td class="font-monospace small text-muted">' +
              escapeHtml(shortID(t.id)) +
              "</td></tr>"
            );
          })
          .join("");
      }
    }

    var prevDisabled = page <= 1;
    var nextDisabled = totalPages === 0 || page >= totalPages;
    var prevBtn = prevDisabled
      ? '<button type="button" class="btn btn-outline-secondary btn-sm" disabled>Previous</button>'
      : '<button type="button" class="btn btn-outline-secondary btn-sm" hx-get="' +
        apiTxnURL(accountID, page - 1, limit, filterType) +
        '" hx-target="#txn-panel" hx-swap="none" hx-headers=\'{"Accept":"application/json"}\'>Previous</button>';
    var nextBtn = nextDisabled
      ? '<button type="button" class="btn btn-outline-secondary btn-sm" disabled>Next</button>'
      : '<button type="button" class="btn btn-outline-secondary btn-sm" hx-get="' +
        apiTxnURL(accountID, page + 1, limit, filterType) +
        '" hx-target="#txn-panel" hx-swap="none" hx-headers=\'{"Accept":"application/json"}\'>Next</button>';

    var pagination = document.getElementById("txn-pagination");
    if (pagination) {
      pagination.innerHTML =
        '<nav aria-label="Transaction pagination" class="d-flex justify-content-between align-items-center flex-wrap gap-2 mt-3">' +
        '<div class="text-muted small font-monospace" id="pagination-meta">' +
        "<span>current_page: " +
        pag.current_page +
        '</span><span class="mx-1">·</span>' +
        "<span>page_size: " +
        pag.page_size +
        '</span><span class="mx-1">·</span>' +
        "<span>total_records: " +
        pag.total_records +
        '</span><span class="mx-1">·</span>' +
        "<span>total_pages: " +
        pag.total_pages +
        "</span></div><div class=\"btn-group\">" +
        prevBtn +
        nextBtn +
        "</div></nav>";
      if (window.htmx) htmx.process(pagination);
    }
  }

  function refreshBalance(accountID) {
    if (!accountID) return;
    fetch("/api/accounts/" + encodeURIComponent(accountID) + "/balance", {
      headers: { Accept: "application/json" },
    })
      .then(function (res) {
        return res.json().then(function (body) {
          appendJsonLog({
            method: "GET",
            path: "/api/accounts/" + accountID + "/balance",
            status: res.status,
            errorCode: "",
            body: body,
          });
          if (body && body.success && body.data) {
            var panel = document.getElementById("txn-panel");
            if (panel) panel.setAttribute("data-balance", String(body.data.balance));
            var balEl = document.getElementById("account-balance");
            if (balEl) balEl.textContent = formatMoney(body.data.balance);
          }
        });
      })
      .catch(function () {});
  }

  function handleApiJson(detail) {
    if (!detail || !detail.xhr) return;
    var path = requestPath(detail);
    var xhr = detail.xhr;
    var body = tryParseJSON(xhr.responseText);
    if (!body) return;
    if (!isApiPath(path) && !(detail.target && detail.target.id && /alert/.test(detail.target.id))) {
      return;
    }

    appendJsonLog({
      method: (detail.requestConfig && detail.requestConfig.verb) || "REQUEST",
      path: path || "/api",
      status: xhr.status,
      errorCode: (body.error && body.error.code) || xhr.getResponseHeader("X-Error-Code") || "",
      body: body,
    });

    if (detail.target && detail.target.id && /alert/.test(detail.target.id)) {
      renderStatusAlert(detail.target, xhr.status, body);
    }

    if (isTxnListPayload(body)) {
      renderTxnPanelFromAPI(path, body);
    }
  }

  document.body.addEventListener("htmx:configRequest", function (evt) {
    var elt = evt.detail.elt;
    var path = "";
    if (elt && elt.getAttribute) {
      path = elt.getAttribute("hx-post") || elt.getAttribute("hx-get") || "";
    }
    if (!path) path = evt.detail.path || "";
    if (isApiPath(path)) {
      evt.detail.headers["Accept"] = "application/json";
    }
  });

  document.body.addEventListener("htmx:beforeSwap", function (evt) {
    var body = tryParseJSON(evt.detail.xhr && evt.detail.xhr.responseText);
    var path = requestPath(evt.detail);
    if (body && (isApiPath(path) || (evt.detail.target && evt.detail.target.id && /alert/.test(evt.detail.target.id)))) {
      evt.detail.shouldSwap = false;
      evt.detail.isError = false;
    }
  });

  document.body.addEventListener("htmx:afterRequest", function (evt) {
    handleApiJson(evt.detail);
    if (!evt.detail || !evt.detail.xhr) return;
    var xhr = evt.detail.xhr;
    if (xhr.status >= 200 && xhr.status < 300 && xhr.getResponseHeader("HX-Trigger")) {
      try {
        var trigger = JSON.parse(xhr.getResponseHeader("HX-Trigger"));
        if (trigger.transactionCreated || trigger.transferCreated) {
          var panel = document.getElementById("txn-panel");
          var accountID = (panel && panel.getAttribute("data-account-id")) || trigger.accountId || "";
          if (typeof trigger.balance !== "undefined" && panel) {
            panel.setAttribute("data-balance", String(trigger.balance));
          } else if (accountID) {
            refreshBalance(accountID);
          }
          // Reload page 1 from backend API after mutations.
          if (panel && accountID && window.htmx) {
            var limit = panel.getAttribute("data-limit") || "3";
            var filter = panel.getAttribute("data-filter") || "";
            htmx.ajax("GET", apiTxnURL(accountID, 1, limit, filter), {
              target: "#txn-panel",
              swap: "none",
              headers: { Accept: "application/json" },
            });
          } else {
            document.body.dispatchEvent(new Event("refresh-txns"));
          }
        }
      } catch (_) {}
    }
  });

  document.addEventListener("DOMContentLoaded", function () {
    if (localStorage.getItem(STORAGE_KEY) === null) {
      setPanelOpen(true);
    } else {
      setPanelOpen(isPanelOpen());
    }

    var clearBtn = document.getElementById("clear-json-log");
    if (clearBtn) {
      clearBtn.addEventListener("click", function () {
        var log = document.getElementById("json-status-log");
        if (log) log.innerHTML = "";
      });
    }

    var collapse = document.getElementById("collapse-json-panel");
    if (collapse) collapse.addEventListener("click", function () { setPanelOpen(false); });

    var fab = document.getElementById("json-panel-fab");
    if (fab) {
      fab.addEventListener("click", function () {
        setPanelOpen(true);
        var log = document.getElementById("json-status-log");
        if (log) log.scrollTop = log.scrollHeight;
      });
    }

    fetch("/api/accounts?page=1&limit=3", { headers: { Accept: "application/json" } })
      .then(function (res) {
        return res.json().then(function (body) {
          appendJsonLog({
            method: "GET",
            path: "/api/accounts?page=1&limit=3",
            status: res.status,
            errorCode: "",
            body: body,
          });
        });
      })
      .catch(function (err) {
        appendJsonLog({
          method: "GET",
          path: "/api/accounts",
          status: 0,
          errorCode: "CLIENT_ERROR",
          body: { success: false, error: { message: String(err) } },
        });
      });
  });
})();
