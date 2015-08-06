(function(){
"use strict";

var onready = (function(){
    var loaded = document.readyState == "complete";
    var loadHandlers = [];
    if ( !loaded ) {
        document.addEventListener("DOMContentLoaded", function () {
            loaded = true;
            for (var i = 0; i < loadHandlers.length; i++)
                loadHandlers[i]();
            loadHandlers = [];
        });
    }

    return function (fn) {
        if ( loaded ) {
            fn();
        } else {
            loadHandlers.push(fn);
        }
    };
})();

var naturalCompare = function (a, b) {
    a = a + "";
    b = b + "";

    var origA = a;
    var origB = b;

    a = a.toLowerCase();
    b = b.toLowerCase();

    var numRe = /^[0-9]+/;

    while ( a.length > 0 && b.length > 0 ) {
        var am = numRe.exec(a);
        var bm = numRe.exec(b);

        if ( am && bm ) {
            // if a number starts at the same location, compare it numerically
            var anum = parseInt(am[0], 10);
            var bnum = parseInt(bm[0], 10);
            if ( anum < bnum ) return -1;
            if ( anum > bnum ) return 1;
        }

        // compare leading characters
        var ac = a.charCodeAt(0);
        var bc = b.charCodeAt(0);
        if ( ac < bc ) return -1;
        if ( ac > bc ) return 1;

        // step one character forward
        a = a.substring(1);
        b = b.substring(1);
    }

    return a.length < b.length ? -1 : a.length > b.length ? 1 : // longer comes later
        origA < origB ? -1 : origA > origB ? 1 : 0; // finally, fall through to case-sensitive comparison
};

var onResize = function (ev) {
    render();
};

var onKeyDown = function (ev) {
    if ( !state.files )
        return;
    if ( ev.keyCode == 37 ) {
        ev.preventDefault();
        left();
    } else if ( ev.keyCode == 39 ) {
        ev.preventDefault();
        right();
    }
};

var onClick = function (ev) {
    if ( !state.files )
        return;
    ev.preventDefault();
    if ( ev.clientX > window.innerWidth/2 ) {
        right();
    } else {
        left();
    }
};

var render = function () {
    var el;
    while ( el = state.rootEl.firstChild )
        state.rootEl.removeChild(el);

    state.rootEl.style.height = window.innerHeight+"px";

    if ( state.error ) {
        renderText(state.error);
        return;
    }

    if ( !state.files ) {
        renderText("Opening "+state.zipURL);
        return;
    }

    var ctxMsg = "[" + (state.index+1) + "/" + state.files.length + "] ";
    ctxMsg += state.files[state.index].filename;
    if ( state.loading )
        ctxMsg += " (loading)";

    var ctx = document.createElement("div");
    ctx.classList.add("viewerContext");
    ctx.appendChild(document.createTextNode(ctxMsg));
    if ( state.loading )
        ctx.classList.add("loading");

    state.rootEl.appendChild(ctx);

    if ( state.imageError ) {
        renderText("Couldn't open "+state.files[state.index].filename+" in zip file: "+state.imageError);
        return;
    }

    if ( state.image ) {
        state.rootEl.appendChild(state.image);
        return;
    }

    renderText("Loading...");
};

var renderText = function (text) {
    var msg = document.createElement("div");
    msg.classList.add("viewerMessage");
    msg.appendChild(document.createTextNode(text));
    state.rootEl.appendChild(msg);
};

var beginWithFiles = function (files) {
    files.sort(function (a, b) {
        return naturalCompare(a.filename, b.filename);
    });

    state.files = files;
    state.index = 0;

    switchImage(state.index);
};

var left = function () {
    if ( state.index == 0 )
        return;
    state.index--;
    switchImage(state.index);
};

var right = function () {
    if ( state.index == state.files.length-1 )
        return;
    state.index++;
    switchImage(state.index);
};

var switchImage = function (index) {
    state.loading = true;

    var didRender = false;
    loadImage(index, function (img, err) {
        if ( state.index == index ) {
            state.loading = false;
            state.image = img;
            state.imageError = err;
            didRender = true;
            render();
            setTimeout(function () {
                if ( index > 0 )
                    loadImage(index-1, function () {});
                if ( index < state.files.length-1 )
                    loadImage(index+1, function () {});
            }, 100);
        }
    });

    if ( !didRender )
        render();

    for (var k in state.imageCache) {
        var i = parseInt(k, 10);
        if ( i < index-2 || i > index+2 ) {
            console.log("expiring entry "+i);
            delete state.imageCache[k];
        }
    }
};

var loadImage = function (index, cb) {
    var entry = state.imageCache[index];
    if ( entry ) {
        if ( entry.done ) {
            cb(entry.image, entry.error);
        } else {
            entry.cbs.push(cb);
        }
        return;
    }

    var entry = {
        cbs: [cb],
        image: null,
        error: null,
        done: false,
    };
    state.imageCache[index] = entry;

    console.log("loadImage start on ["+index+"] "+state.files[index].filename);

    state.files[index].getData(new zip.Data64URIWriter(), function (url) {
        console.log("loadImage success on ["+index+"] "+state.files[index].filename);
        entry.image = document.createElement('img');
        entry.image.src = url;
        entry.done = true;
        for (var i = 0; i < entry.cbs.length; i++)
            entry.cbs[i](entry.image, entry.error);
    });
};

var state = {
    rootEl: null,
    zipURL: null,
    error: null,
    imageError: null,
    image: null,
    files: null,
    index: 0,
    imageCache: {},
    lastLoadedIndex: 0,
};

onready(function () {
    zip.useWebWorkers = false;

    var rootEl = state.rootEl = document.createElement("div");
    rootEl.classList.add("viewerBase");
    document.body.appendChild(rootEl);

    window.addEventListener("resize", onResize, false);
    window.addEventListener("keydown", onKeyDown, false);
    window.addEventListener("click", onClick, false);

    var where = window.location.hash.replace(/^#/, '');

    if ( where == "" ) {
        state.error = "No ZIP URL given";
        render();
        return;
    }

    state.zipURL = where;
    console.log("loading from "+where);

    render();

    zip.createReader(new zip.HttpReader(where), function (r) {
        r.getEntries(function (entries) {
            beginWithFiles(entries);
        });
    }, function (err) {
        state.error = "Couldn't load from "+where+": "+err;
        render();
    });
});

})();
