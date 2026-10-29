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

function findComment(root, data) {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_COMMENT);
  let node;
  while ((node = walker.nextNode())) {
    if (node.nodeValue === data) return node;
  }
  return null;
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
      node.remove();
    }
    for (let i = nodes.length; i < items.length; i++) {
      const node = render(i, items[i]);
      close.parentNode.insertBefore(node, close);
      nodes.push(node);
    }
  });
}

export function list(root, marker, render) {
  const open = findComment(root, `g:each:${marker}:open`);
  const close = findComment(root, `g:each:${marker}:close`);
  if (!open || !close) return () => {};

  const nodes = [];
  for (let node = open.nextSibling; node && node !== close; ) {
    const next = node.nextSibling;
    if (node.nodeType === 1) nodes.push(node);
    node = next;
  }

  return function update(items) {
    while (nodes.length > items.length) {
      nodes.pop().remove();
    }
    for (let i = nodes.length; i < items.length; i++) {
      const node = render(items[i], i);
      close.parentNode.insertBefore(node, close);
      nodes.push(node);
    }
  };
}

export function on(root, marker, event, handler) {
  const el = root.querySelector(`[data-gothic-on="${marker}"]`);
  if (!el) return;
  el.removeAttribute("data-gothic-on");
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

export function readState(id) {
  const el = document.querySelector(`script[data-gothic-state="${id}"]`);
  if (!el) return {};
  try {
    return JSON.parse(el.textContent || "{}");
  } catch (_) {
    return {};
  }
}
