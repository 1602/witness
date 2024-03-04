let connection;
let openRequests = new Map();
let connected;
let reconnectingTimeout = null;

function connect() {
    if (reconnectingTimeout) {
        clearTimeout(reconnectingTimeout);
    }
    if (connection && connection.readyState === 2 || !connection) {
        connection = new EventSource('http://localhost:8989/events');
    }
    connection.onmessage = (e) => {
        localStorage.lastMessage = e.data;
        try {
            handle(JSON.parse(e.data));
        } catch (e) {
            log.error('handle error', e);
        }
    };

    // connection.addEventListener('close-app', onClose);

    connection.onopen = () => {
        updateConnected(true);
    };

    connection.onerror = onClose;

    function onClose(e) {
        updateConnected(false);
        connection.close();
        clearTimeout(reconnectingTimeout);
        reconnectingTimeout = setTimeout(connect, 10000);
    }
}

document.addEventListener("visibilitychange", (evt) => {
    if (!document.hidden || ['focus', 'focusin', 'pageshow'].includes(evt.type)) {
        console.log('visibilitychange', evt.type, document.hidden);
        connect();
    }
});

let app;
let header;
let details;

document.addEventListener('beforeunload', () => {
    connection.close();
});

if (document.readyState === 'complete') {
    init();
} else {
    document.addEventListener('DOMContentLoaded', () => init());
}


function init() {
    header = document.createElement('div');
    header.className = 'header';
    document.body.appendChild(header);

    updateConnected(false);

    app = document.createElement('div');
    app.className = 'logs';
    app.style.height = '400px';
    document.body.appendChild(app);

    const divider = document.createElement('div');
    divider.className = 'divider';
    divider.addEventListener('mousedown', (e) => {
        document.body.addEventListener('mousemove', onMouseMove);
        document.body.addEventListener('mouseup', onMouseUp);

        const initialScreenY = e.screenY;
        const initialHeight = app.offsetHeight;
        const detailsInitialHeight = details.offsetHeight;

        function onMouseUp() {
            document.body.removeEventListener('mousemove', onMouseMove);
            document.body.removeEventListener('mouseup', onMouseUp);
        }

        function onMouseMove(e) {
            const delta = e.screenY - initialScreenY;
            app.style.height = `${initialHeight + delta}px`;
            details.style.height = `${detailsInitialHeight - 4 - delta}px`;
        }
    });
    document.body.appendChild(divider);

    details = document.createElement('div');
    details.className = 'details';
    details.style.height = `${document.body.offsetHeight - 400 - header.offsetHeight - divider.offsetHeight - 6}px`;
    document.body.appendChild(details);

    // debug
    if (localStorage.lastMessage) {
        // handle(JSON.parse(localStorage.lastMessage));
    }

    connect();
}

function updateConnected(newValue) {
    if (newValue === connected) {
        return;
    }
    connected = newValue;
    header.innerText = connected ? 'connected' : 'connecting...';
}

function handle(data) {
    console.log(data);
    logRequest(data);
}

function logRequest(rt) {
    const { id, done } = rt;
    let log = openRequests.get(id);
    if (log) {
        if (!log.rt.error && rt.error) {
            log.el.classList.add('error');
        }
        log.rt = rt;
        log.el.sourceData = rt;
        render(log);
    } else {
        const el = document.createElement('div');
        el.className = 'log';
        if (rt.error) {
            el.className += ' error';
        }
        log = { el, rt };
        openRequests.set(id, log);
        app.appendChild(el);
        el.sourceData = rt;

        render(log);
    }

    updateWaterfall();

    if (done) {
        openRequests.delete(id);
    }
}

function updateWaterfall() {
    const logs = document.querySelector('.logs');
    if (!logs) {
        return;
    }

    let logEl = logs.firstChild;
    if (!logEl) {
        return;
    }

    const startDate = new Date(logEl.sourceData.timeline.startedAt).valueOf();
    const endDate  = new Date(logs.lastChild.sourceData.timeline.startedAt).valueOf() + logs.lastChild.sourceData.durationNano / 1_000_000;

    if (startDate === endDate) {
        return;
    }

    const totalDuration = endDate - startDate;

    while (logEl) {
        const tick = logEl.firstChild.lastChild;
        const eventStart = new Date(logEl.sourceData.timeline.startedAt).valueOf();
        const relativeStart = eventStart - startDate;

        tick.style.left = (relativeStart / totalDuration * 100) + '%';
        tick.style.width = (logEl.sourceData.durationNano / 1_000_000) / totalDuration * 100 + '%';

        logEl = logEl.nextSibling;
    }
}

function render(log) {
    const {
        requestLog: req,
        responseLog: res,
        timeline,
        done,
        duration,
        error
    } = log.rt;

    log.el.innerHTML = `<div class="row">${req.method} ${req.url} ${ status(res) } ${ res ? formatByteLen(res.contentLength) : '' } ${ duration } ${ error ? error.message : '' }<span class="waterfall"></span></div>`;

    log.el.firstChild.addEventListener('click', () => {
        details.innerHTML = `<div> ${ expanded(log) }</div>`;
    });
}

function status(res) {
    if (!res) {
        return '';
    }

    const statusClass = Math.floor(res.statusCode / 100);

    return `<span class="status-${statusClass}xx">${res.status}</span>`;
}

function expanded(log) {
    const {
        requestLog: req,
        responseLog: res,
        error,
    } = log.rt;

    const body = res && res.body ?
        `response body: <pre class="json"> ${ JSON.stringify(JSON.parse(res.body), ' ', 4) } </pre>` : '';

    const err = error ? `error details: <pre>${ JSON.stringify(error.details, ' ', 4) }</pre>` : '';

    const timeline = log.rt.timeline.events.map(e => {
        return `<div>${ e.name } - ${ (e.delay / 1000000).toFixed(1) }ms ${ payload(e) }</div>`
    });
    timeline.push(`<div><hr/>end: ${log.rt.duration}</div>`);

    return `
        <div>
            ${ body }
            ${ err }
            timeline: ${ timeline.join('') }
        </div>
    `;
}

function payload(e) {
    if (e.name === 'GotConn') {
        if (e.payload.Reused) {
            if (e.payload.WasIdle) {
                return 'reused, idle for ' + e.payload.IdleTime;
            } else {
                return 'reused, not idle';
            }
        } else {
            return 'new';
        }
    } else if (e.name === 'WroteHeaderField') {
        return `${ e.payload.key }: ${ e.payload.value.join(', ')}`;
    }

    return '';
}

function formatByteLen(n) {
    if (n < (1000)) {
        return n.toString() + 'B';
    } else if (n < (1000000)) {
        return (n / (1000)).toFixed(2) + 'kB';
    } else if (n < (1000000000)) {
        return (n / (1000000)).toFixed(2) + 'MB';
    } else {
        return (n / (1000000000)).toFixed(2) + 'GB';
    }
}
