// gothic.js is the iteration 1 client runtime. It is embedded in the Go binary
// and served at /_gothic/runtime.js. It contains signals, a small effect
// scheduler, DOM adoption helpers, and the JSON procedure transport.

let activeEffect = null;
let scheduled = false;
const pending = new Set();

export function signal(initial) {
  let value = initial;
  const subscribers = new Set();

  function read() {
    if (activeEffect) {
      subscribers.add(activeEffect);
      activeEffect.deps.push(subscribers);
    }
    return value;
  }

  read.set = (next) => {
    if (Object.is(next, value)) return;
    value = next;
    for (const subscriber of [...subscribers]) pending.add(subscriber);
    schedule();
  };

  read.push = (item) => {
    read.set([...value, item]);
  };

  return read;
}

export function effect(fn) {
  const e = { deps: [], run: null };
  e.run = () => {
    for (const dep of e.deps) dep.delete(e);
    e.deps = [];
    const previous = activeEffect;
    activeEffect = e;
    try {
      fn();
    } finally {
      activeEffect = previous;
    }
  };
  e.run();
  return e;
}

function schedule() {
  if (scheduled) return;
  scheduled = true;
  queueMicrotask(() => {
    scheduled = false;
    const effects = [...pending];
    pending.clear();
    for (const e of effects) e.run();
  });
}

function findComment(root, data, skipComponents) {
  function walk(node) {
    for (let child = node.firstChild; child; child = child.nextSibling) {
      if (child.nodeType === 8 && child.nodeValue === data) return child;
      if (child.nodeType === 1) {
        if (skipComponents && child.hasAttribute("data-gothic-component")) continue;
        const found = walk(child);
        if (found) return found;
      }
    }
    return null;
  }
  return walk(root);
}

function findEl(root, selector, skipComponents) {
  const candidates = [];
  if (root.matches?.(selector)) candidates.push(root);
  if (root.querySelectorAll) candidates.push(...root.querySelectorAll(selector));
  for (const el of candidates) {
    if (!skipComponents) return el;
    const owner = el.closest?.("[data-gothic-component]");
    if (!owner || owner === root) return el;
  }
  return null;
}

function destroyComponentTree(node) {
  if (!node || node.nodeType !== 1) return;
  const nodes = [];
  if (node.matches && node.matches("[data-gothic-component]")) nodes.push(node);
  if (node.querySelectorAll) nodes.push(...node.querySelectorAll("[data-gothic-component]"));
  for (const n of nodes) {
    const instances = n.__gothicInstances;
    if (!instances) continue;
    for (const inst of Object.values(instances)) {
      if (inst && typeof inst.destroy === "function") inst.destroy();
    }
    n.__gothicInstances = null;
  }
}

export function each(root, marker, sig, render) {
  const open = findComment(root, `g:each:${marker}:open`);
  const close = findComment(root, `g:each:${marker}:close`);
  if (!open || !close) return;

  const nodes = [];
  for (let node = open.nextSibling; node && node !== close; ) {
    const next = node.nextSibling;
    if (node.nodeType === 1) nodes.push(node);
    node = next;
  }

  effect(() => {
    const items = sig();
    while (nodes.length > items.length) {
      const node = nodes.pop();
      destroyComponentTree(node);
      node.remove();
    }
    for (let i = nodes.length; i < items.length; i++) {
      const node = render(i, items[i]);
      close.parentNode.insertBefore(node, close);
      nodes.push(node);
    }
  });
}

export function list(root, marker, render, hydrate, key, skipComponents = true) {
  const open = findComment(root, `g:each:${marker}:open`, skipComponents);
  const close = findComment(root, `g:each:${marker}:close`, skipComponents);
  if (!open || !close) return () => {};

  const nodes = [];
  for (let node = open.nextSibling; node && node !== close; ) {
    const next = node.nextSibling;
    if (node.nodeType === 1) nodes.push(node);
    node = next;
  }

  let hydrated = false;
  return function update(items) {
    items = items || [];
    if (key) {
      const byKey = new Map();
      for (const node of nodes) byKey.set(node.getAttribute("data-gothic-key"), node);

      const next = [];
      for (let i = 0; i < items.length; i++) {
        const itemKey = String(key(items[i], i));
        let node = byKey.get(itemKey);
        if (node) {
          byKey.delete(itemKey);
          if (hydrate) hydrate(node, items[i], i);
        } else {
          node = render(items[i], i);
        }
        next.push(node);
      }
      for (const node of byKey.values()) {
        destroyComponentTree(node);
        node.remove();
      }

      let ref = close;
      for (let i = next.length - 1; i >= 0; i--) {
        const node = next[i];
        if (node.nextSibling !== ref) {
          close.parentNode.insertBefore(node, ref);
        }
        ref = node;
      }
      nodes.length = 0;
      nodes.push(...next);
      return;
    }

    if (!hydrated) {
      hydrated = true;
      if (hydrate) {
        for (let i = 0; i < nodes.length && i < items.length; i++) {
          hydrate(nodes[i], items[i], i);
        }
      }
    }
    while (nodes.length > items.length) {
      const node = nodes.pop();
      destroyComponentTree(node);
      node.remove();
    }
    for (let i = nodes.length; i < items.length; i++) {
      const node = render(items[i], i);
      close.parentNode.insertBefore(node, close);
      nodes.push(node);
    }
  };
}

export function text(root, marker, value, skipComponents = true) {
  const open = findComment(root, `g:text:${marker}:open`, skipComponents);
  const close = findComment(root, `g:text:${marker}:close`, skipComponents);
  if (!open || !close) return;

  let node = open.nextSibling;
  while (node && node !== close && node.nodeType !== 3) {
    node = node.nextSibling;
  }
  if (!node || node === close) {
    node = document.createTextNode("");
    close.parentNode.insertBefore(node, close);
  }
  node.nodeValue = String(value);
}

export function on(root, marker, event, handler, skipComponents = true) {
  const selector = `[data-gothic-on~="${marker}"]`;
  const el = findEl(root, selector, skipComponents);
  if (!el) return;

  const markers = (el.getAttribute("data-gothic-on") || "")
    .split(/\s+/)
    .filter((value) => value && value !== String(marker));
  if (markers.length > 0) {
    el.setAttribute("data-gothic-on", markers.join(" "));
  } else {
    el.removeAttribute("data-gothic-on");
  }
  el.addEventListener(event, handler);
}

export async function rpc(name, ...args) {
  const res = await fetch(`/_gothic/rpc/${encodeURIComponent(name)}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ args }),
  });
  let data = null;
  try {
    data = await res.json();
  } catch (_) {
    data = null;
  }
  if (!res.ok || (data && data.error)) {
    const message =
      data && data.error && data.error.message
        ? data.error.message
        : `rpc ${name} failed`;
    throw new Error(message);
  }
  return data ? data.result : undefined;
}

export function len(value) {
  if (value == null) return 0;
  if (Array.isArray(value) || typeof value === "string") return value.length;
  if (typeof value === "object") return Object.keys(value).length;
  return 0;
}

export function entries(value) {
  if (value == null) return [];
  if (Array.isArray(value)) return value.map((item, index) => [index, item]);
  if (typeof value === "object") return Object.entries(value);
  return [];
}

export function readState(id) {
  const el = document.querySelector(`script[data-gothic-state="${id}"]`);
  if (!el) return {};
  try {
    return JSON.parse(el.textContent || "{}");
  } catch (_) {
    return {};
  }
}

export function attr(root, marker, value, skipComponents = true) {
  const key = `data-gothic-attr-${marker}`;
  const selector = `[${key}]`;
  const el = findEl(root, selector, skipComponents);
  if (!el) return;

  const name = el.getAttribute(key);
  if (name === "checked") {
    el.checked = !!value;
    return;
  }
  if (name === "value") {
    const next = value == null ? "" : String(value);
    if (el.value !== next) el.value = next;
    return;
  }
  if (value === false || value == null) {
    el.removeAttribute(name);
  } else {
    el.setAttribute(name, String(value));
  }
}

export function classToggle(root, marker, active, skipComponents = true) {
  const key = `data-gothic-class-${marker}`;
  const selector = `[${key}]`;
  const el = findEl(root, selector, skipComponents);
  if (!el) return;

  const name = el.getAttribute(key);
  if (name) el.classList.toggle(name, !!active);
}

export function focus(root, marker, active, skipComponents = true) {
  const selector = `[data-gothic-focus="${marker}"]`;
  const el = findEl(root, selector, skipComponents);
  if (!el) return;

  if (active) {
    if (!el.__gothicFocused) {
      el.__gothicFocused = true;
      setTimeout(() => {
        if (el.__gothicFocused) el.focus();
      }, 0);
    }
  } else {
    if (el.__gothicFocused) el.blur();
    el.__gothicFocused = false;
  }
}


const ifControllers = new WeakMap();

function ifServerBranch(open, marker) {
  const node = open.nextSibling;
  if (!node || node.nodeType !== 8) return null;
  if (node.nodeValue === `g:if:${marker}:then`) return "then";
  if (node.nodeValue === `g:if:${marker}:else`) return "else";
  return null;
}

function ifClear(open, close) {
  let node = open.nextSibling;
  while (node && node !== close) {
    const next = node.nextSibling;
    destroyComponentTree(node);
    node.remove();
    node = next;
  }
}

function ifBlockInternal(root, marker, test, buildThen, buildElse, bindThen, bindElse, skipComponents) {
  const open = findComment(root, `g:if:${marker}:open`, skipComponents);
  const close = findComment(root, `g:if:${marker}:close`, skipComponents);
  if (!open || !close) return;

  let state = ifControllers.get(open);
  if (!state) {
    state = { branch: null, started: false };
    ifControllers.set(open, state);
  }

  const want = test ? "then" : "else";

  if (!state.started) {
    state.started = true;
    const rendered = ifServerBranch(open, marker);
    if (rendered) {
      state.branch = rendered;
    }
  }

  if (state.branch === want) {
    (want === "then" ? bindThen : bindElse)(open.parentNode);
    return;
  }

  ifClear(open, close);
  close.parentNode.insertBefore(document.createComment(`g:if:${marker}:${want}`), close);
  const frag = want === "then" ? buildThen() : buildElse();
  close.parentNode.insertBefore(frag, close);
  state.branch = want;
  (want === "then" ? bindThen : bindElse)(open.parentNode);
}

export function ifBlock(root, marker, test, buildThen, buildElse, bindThen, bindElse) {
  return ifBlockInternal(root, marker, test, buildThen, buildElse, bindThen, bindElse, true);
}

export function ifBlockLocal(root, marker, test, buildThen, buildElse, bindThen, bindElse) {
  return ifBlockInternal(root, marker, test, buildThen, buildElse, bindThen, bindElse, false);
}

export function textLocal(root, marker, value) { return text(root, marker, value, false); }
export function onLocal(root, marker, event, handler) { return on(root, marker, event, handler, false); }
export function attrLocal(root, marker, value) { return attr(root, marker, value, false); }
export function classToggleLocal(root, marker, active) { return classToggle(root, marker, active, false); }
export function focusLocal(root, marker, active) { return focus(root, marker, active, false); }
export function listLocal(root, marker, render, hydrate, key) { return list(root, marker, render, hydrate, key, false); }
