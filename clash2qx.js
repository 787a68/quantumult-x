/**
 * @fileoverview Clash-to-QX resource parser: converts anytls nodes from
 *                Clash subscription content to Quantumult X server format.
 *
 * @supported Quantumult X (v1.0.10-build277+)
 *
 * Usage in QX [server_remote]:
 *   https://316.sub987.top/sub, tag=Sub987, opt-parser=true
 *
 * Set subscription User-Agent to "ClashMeta" in QX so the endpoint
 * returns Clash config format. This parser only does the conversion.
 */

var BODY = $resource && $resource.content ? $resource.content : '';

if (!BODY) {
    $done({ error: 'empty response body' });
    return;
}

// ---------- Pre-compiled regexes ----------
var INLINE_RE = /-\s*\{([\s\S]*?)\}/g;
var BLOCK_SPLIT_RE = /^\-\s*(?!\{)/gm;
var KV_LINE_RE = /^\s*([a-zA-Z0-9\-_]+):\s*(.*)$/;

// ---------- Utility functions ----------
function cleanString(v) {
    if (v === undefined || v === null) return '';
    var s = String(v).trim();
    if (!s) return '';
    var c0 = s.charAt(0), c1 = s.charAt(s.length - 1);
    if ((c0 === "'" && c1 === "'") || (c0 === '"' && c1 === '"')) {
        s = s.slice(1, -1).trim();
    }
    if (s === "''" || s === '""') return '';
    return s;
}

function parseInlineObject(text) {
    var obj = Object.create(null);
    var parts = text.split(/,(?=(?:[^'"]|'[^']*'|"[^"]*")*$)/);
    for (var i = 0; i < parts.length; i++) {
        var p = parts[i];
        var idx = p.indexOf(':');
        if (idx === -1) continue;
        var k = p.slice(0, idx).trim();
        var v = p.slice(idx + 1).trim();
        if (!v) { obj[k] = ''; continue; }
        var fc = v.charAt(0), lc = v.charAt(v.length - 1);
        if ((fc === "'" && lc === "'") || (fc === '"' && lc === '"')) v = v.slice(1, -1);
        obj[k] = v;
    }
    return obj;
}

function parseBlockObject(blockText) {
    var obj = Object.create(null);
    var lines = blockText.split('\n');
    for (var i = 0; i < lines.length; i++) {
        var line = lines[i];
        var m = line.match(KV_LINE_RE);
        if (!m) continue;
        var k = m[1];
        var v = m[2] === undefined ? '' : m[2].trim();
        if (v === '|' || v === '>') {
            var j = i + 1;
            var buf = [];
            while (j < lines.length && /^\s+/.test(lines[j])) {
                buf.push(lines[j].replace(/^\s+/, ''));
                j++;
            }
            v = buf.join('\n');
            i = j - 1;
        } else {
            var fc = v.charAt(0), lc = v.charAt(v.length - 1);
            if ((fc === "'" && lc === "'") || (fc === '"' && lc === '"')) v = v.slice(1, -1);
        }
        obj[k] = v;
    }
    return obj;
}

function parseBooleanField(raw) {
    if (raw === undefined || raw === null) return { present: false, value: null };
    var s = String(raw).trim().toLowerCase();
    if (s === '') return { present: true, value: null };
    if (s === 'true' || s === '1' || s === 'yes') return { present: true, value: true };
    if (s === 'false' || s === '0' || s === 'no') return { present: true, value: false };
    return { present: true, value: null };
}

// ---------- Protocol handlers ----------
// Each handler receives the parsed raw object from Clash config and returns
// an array of QX server lines. Register new protocols via:
//   handlers.<type> = function(raw) { ... return [...lines]; };
// or:
//   registerHandler('<type>', function(raw) { ... });
var handlers = Object.create(null);

handlers.anytls = function (raw) {
    var server = cleanString(raw.server);
    if (!server) return [];
    var port = cleanString(raw.port);
    var password = cleanString(raw.password);
    var sni = cleanString(raw.sni);
    var pubkey = cleanString(raw['reality-base64-pubkey']);
    var shortid = cleanString(raw['reality-hex-shortid']);
    var name = cleanString(raw.name);
    var udpInfo = parseBooleanField(raw.udp);

    var overRaw = raw['over-tls'] !== undefined
        ? raw['over-tls']
        : (raw['over_tls'] !== undefined ? raw['over_tls'] : undefined);
    var overInfo = parseBooleanField(overRaw);
    var overPresent = overInfo.present;
    var overValue = overInfo.value;
    if (!overPresent || overValue === null) { overPresent = true; overValue = true; }

    var hostPort = port ? (server + ':' + port) : server;
    var out = [];

    var partsStd = [];
    partsStd.push('anytls=' + hostPort);
    if (password) partsStd.push('password=' + password);
    partsStd.push('over-tls=' + (overValue ? 'true' : 'false'));
    if (sni) partsStd.push('tls-host=' + sni);
    if (udpInfo.present && udpInfo.value === true) partsStd.push('udp-relay=true');
    if (name) partsStd.push('tag=' + name);
    out.push(partsStd.join(', '));

    if (pubkey) {
        var partsReal = [];
        partsReal.push('anytls=' + hostPort);
        if (password) partsReal.push('password=' + password);
        partsReal.push('over-tls=' + (overValue ? 'true' : 'false'));
        if (sni) partsReal.push('tls-host=' + sni);
        partsReal.push('reality-base64-pubkey=' + pubkey);
        if (shortid) partsReal.push('reality-hex-shortid=' + shortid);
        if (udpInfo.present && udpInfo.value === true) partsReal.push('udp-relay=true');
        if (name) partsReal.push('tag=' + name);
        out.push(partsReal.join(', '));
    }

    return out;
};

function registerHandler(typeName, fn) {
    if (!typeName || typeof fn !== 'function') return false;
    handlers[typeName] = fn;
    return true;
}

// ---------- Stub handlers for future extension ----------
// Uncomment and implement to support more Clash proxy types.
// Each handler receives a raw object parsed from Clash YAML and must return
// an array of QX server configuration strings.

// handlers.ss = function (raw) {
//     // Clash fields: server, port, cipher, password, udp, name,
//     //               plugin-opts: {mode, host, path, tls}, tls, sni,
//     //               reality-opts: {public-key, short-id}
//     // QX output: shadowsocks=host:port, method=cipher, password=pwd,
//     //            obfs=http|tls|ws|wss|over-tls, obfs-host=..., obfs-uri=...,
//     //            over-tls=true, tls-host=..., udp-relay=true, tag=name
//     return [];
// };

// handlers.vmess = function (raw) {
//     // Clash fields: server, port, uuid, alterId, cipher, udp, tls, sni,
//     //               network: ws|tcp, ws-opts: {path, headers}, name
//     // QX output: vmess=host:port, method=none|aes-128-gcm|chacha20-poly1305,
//     //            password=uuid, aead=true, obfs=ws|over-tls, obfs-host=...,
//     //            obfs-uri=..., over-tls=true, tls-host=..., tag=name
//     return [];
// };

// handlers.vless = function (raw) {
//     // Clash fields: server, port, uuid, udp, tls, sni, flow,
//     //               network: ws|tcp, ws-opts: {path, headers},
//     //               reality-opts: {public-key, short-id}, name
//     // QX output: vless=host:port, method=none, password=uuid,
//     //            obfs=ws|over-tls, obfs-host=..., obfs-uri=...,
//     //            over-tls=true, tls-host=..., vless-flow=xtls-rprx-vision,
//     //            reality-base64-pubkey=..., reality-hex-shortid=..., tag=name
//     return [];
// };

// handlers.trojan = function (raw) {
//     // Clash fields: server, port, password, udp, sni, skip-cert-verify,
//     //               network: ws, ws-opts: {path, headers}, name
//     // QX output: trojan=host:port, password=pwd, over-tls=true,
//     //            tls-host=..., tls-verification=true|false, udp-relay=true,
//     //            obfs=wss, obfs-host=..., obfs-uri=..., tag=name
//     return [];
// };

// handlers.hysteria2 = function (raw) {
//     // Clash fields: server, port, password, obfs, obfs-password,
//     //               sni, skip-cert-verify, udp, name
//     // QX does not natively support hysteria2; would need external plugin.
//     return [];
// };

// ---------- Main parse loop ----------
var out = [];
var m;

while ((m = INLINE_RE.exec(BODY)) !== null) {
    var inner = m[1];
    var rawInline = parseInlineObject(inner);
    var typeInline = cleanString(rawInline.type);
    if (!typeInline) continue;
    var handlerInline = handlers[typeInline];
    if (typeof handlerInline === 'function') {
        var linesInline = handlerInline(rawInline);
        for (var i = 0; i < linesInline.length; i++) out.push(linesInline[i]);
    }
}

var blockIndices = [];
while ((m = BLOCK_SPLIT_RE.exec(BODY)) !== null) blockIndices.push(m.index);

if (blockIndices.length > 0) {
    blockIndices.push(BODY.length);
    for (var i = 0; i < blockIndices.length - 1; i++) {
        var start = blockIndices[i];
        var end = blockIndices[i + 1];
        var block = BODY.slice(start, end);

        var typeMatch = block.match(/^\s*type:\s*([^\n\r]+)/m);
        if (!typeMatch) continue;
        var typeBlock = cleanString(typeMatch[1]);
        if (!typeBlock) continue;

        var handlerBlock = handlers[typeBlock];
        if (typeof handlerBlock !== 'function') continue;

        var rawBlock = parseBlockObject(block);
        if (!rawBlock.type || cleanString(rawBlock.type) !== typeBlock) continue;

        var linesBlock = handlerBlock(rawBlock);
        for (var j = 0; j < linesBlock.length; j++) out.push(linesBlock[j]);
    }
}

if (out.length === 0) {
    $done({ error: 'no anytls nodes found in subscription' });
} else {
    $done({ content: out.join('\n') });
}
