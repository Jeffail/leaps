var ace_editor;

var HSVtoRGB = function(h, s, v) {
	"use strict";

	var r, g, b, i, f, p, q, t;
	if (h && s === undefined && v === undefined) {
		s = h.s, v = h.v, h = h.h;
	}
	i = Math.floor(h * 6);
	f = h * 6 - i;
	p = v * (1 - s);
	q = v * (1 - f * s);
	t = v * (1 - (1 - f) * s);
	switch (i % 6) {
		case 0: r = v, g = t, b = p; break;
		case 1: r = q, g = v, b = p; break;
		case 2: r = p, g = v, b = t; break;
		case 3: r = p, g = q, b = v; break;
		case 4: r = t, g = p, b = v; break;
		case 5: r = v, g = p, b = q; break;
	}
	return {
		r: Math.floor(r * 255),
		g: Math.floor(g * 255),
		b: Math.floor(b * 255)
	};
};

var hash = function(str) {
	"use strict";

	var hash = 0, i, chr, len;
	if ('string' !== typeof str || str.length == 0) {
		return hash;
	}
	for (i = 0, len = str.length; i < len; i++) {
		chr   = str.charCodeAt(i);
		hash  = ((hash << 5) - hash) + chr;
		hash |= 0; // Convert to 32bit integer
	}
	return hash;
};

window.onload = function() {
	"use strict";

	ace_editor = ace.edit("editor");
	ace_editor.setTheme("ace/theme/monokai");
	ace_editor.getSession().setMode("ace/mode/javascript");

	var client = new leap_client();
	client.bind_ace_editor(ace_editor);

	client.on("error", function(err) {
		console.log(JSON.stringify(err));
	});

	client.on("connect", function() {
		client.join_document("anon", "", "test_document");
	});

	client.ACE_set_cursor_handler(function(user_id, session_id, lineHeight, top, left) {
		var height = 40;
		var width = 3;

		var id_hash = hash(session_id);
		if ( id_hash < 0 ) {
			id_hash = id_hash * -1;
		}

		var hue = ( id_hash % 10000 ) / 10000;
		var rgb = HSVtoRGB(hue, 1, 0.8);

		var colorStyle = "rgba(" + rgb.r + ", " + rgb.g + ", " + rgb.b + ", 1)";

		var positionStyle = "";
		var nameBar = "";
		if ( ( top + lineHeight ) < height ) {
			positionStyle = "position: absolute; top: " + top + "px; left: " + left + "px;";
			nameBar = "<div style='position: absolute; top: " + (top + (height - 18) ) +
				"px; left: " + left + "px; background-color: " + colorStyle +
				"; color: #f0f0f0; padding: 4px; font-size: 10px;'>" + user_id.substr(0, 8) + "</div>";
		} else {
			positionStyle = "position: absolute; top: " + ( top - height + lineHeight ) + "px; left: " + left + "px;";
			nameBar = "<div style='" + positionStyle + " background-color: " + colorStyle +
				"; color: #f0f0f0; padding: 4px; font-size: 10px;'>" + user_id.substr(0, 8) + "</div>";
		}

		var markerLine = "<div style='" + positionStyle + " height: " + height + "px; border-left: " + width +
			"px solid " + colorStyle + ";'></div>";

		return markerLine + nameBar;
	});

	client.connect("ws://" + window.location.host + "/socket");
};
