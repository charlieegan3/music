function renderPlays(plays, tableID) {
	var table = document.getElementById(tableID);
	for (var i = 0; i < plays.length; i++) {
		var play = plays[i];
		var row = document.createElement("tr");

		var image = document.createElement("td");
		var img = document.createElement("img");
		img.className = "ba lazy";
		img.setAttribute("style", "min-width: 25px; width: 25px;");
		if (play.Artwork != "") {
			img.setAttribute("data-src", play.Artwork);
		} else {
			img.setAttribute("data-src", "https://upload.wikimedia.org/wikipedia/commons/1/1a/1x1_placeholder.png");
			img.className = "lazy o-0";
		}
		image.appendChild(img);
		row.appendChild(image);

		var track = document.createElement("td");
		track.innerHTML = "<strong>" + play.Track + "</strong> <span class=\"mid-gray\">by</span> " + play.Artist;
		row.appendChild(track);

		if (typeof play.Count != "undefined") {
			var count = document.createElement("td");
			count.innerHTML = "<strong>" + play.Count.toString() + "</strong> plays";
			count.className = "light-silver";
			row.appendChild(count);
		}

		if (typeof play.Timestamp != "undefined") {
			var ts = document.createElement("td");
			ts.innerHTML = timeago().format(play.Timestamp);
			ts.className = "light-silver";
			row.appendChild(ts);
		}

		table.appendChild(row);
	}
}

function renderArtistsWithTracks(artists, trackCount, containerID) {
	var artistsContainer = document.getElementById(containerID);
	artistsContainer.innerHTML = "";

	for (var i = 0; i < artists.length; i++) {
		var header = document.createElement("h3");
		header.innerHTML = artists[i].Name;
		header.className = "f5-ns f6";
		artistsContainer.appendChild(header);

		var trackTable = document.createElement("table");
		trackTable.id = artists[i].Name.replace(/[^\x00-\x7F]/g, "").replace(/ /g, "");
		trackTable.className = "f6-ns f7 w-100";
		artistsContainer.appendChild(trackTable);

		renderPlays(artists[i].Tracks.slice(0, trackCount), trackTable.id);
	}

	new LazyLoad({ elements_selector: ".lazy" });
}

function renderArtists(artists, messageID, count) {
	var list = [];
	for (var i = 0; i < artists.length; i++) {
		list.push(artists[i].Artist)
		if (i >= (count - 1)) {
			break;
		}
	}
	document.getElementById(messageID).innerHTML = list.join(", ");
}

function renderPlaysByMonth(playsByMonth) {
	var months = [], counts = [];
	for (var i = 0; i < playsByMonth.length; i++) {
		months.push(playsByMonth[i].Pretty);
		counts.push(playsByMonth[i].Count);
	}

	var color = Chart.helpers.color;
	var barChartData = {
		labels: months,
		datasets: [{
			data: counts,
			label: "Plays",
			backgroundColor: color("lightgray").alpha(0.5).rgbString(),
			borderColor: "lightgray",
			borderWidth: 1,
		}]
	};

	var ctx = document.getElementById("PlaysByMonth").getContext("2d");
	window.myBar = new Chart(ctx, {
		type: "bar",
		data: barChartData,
		options: {
			responsive: true,
			legend: { display: false },
			title: { display: false },
			scales: {
				xAxes: [{
					gridLines: { display: false }
				}],
				yAxes: [{
					gridLines: { display: false },
					ticks: { beginAtZero: true }
				}]
			}
		}
	});
}
