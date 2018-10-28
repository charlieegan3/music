function renderPlays(plays, tableID) {
	var table = document.getElementById(tableID);
	for (var i = 0; i < plays.length; i++) {
		var play = plays[i];
		var row = document.createElement("tr");

		var image = document.createElement("td");
		var img = document.createElement("img");
		var link = document.createElement("a");
		link.setAttribute("href", "https://open.spotify.com/track/" + play.Spotify);
		link.className = "no-underline dark-gray";
		img.setAttribute("data-src", play.Artwork);
		img.setAttribute("width", 30);
		img.setAttribute("height", 30);
		img.className = "ba lazy";
		link.appendChild(img);
		image.appendChild(link);
		row.appendChild(image);

		var track = document.createElement("td");
		var link = document.createElement("a");
		link.setAttribute("href", "https://open.spotify.com/track/" + play.Spotify);
		link.innerHTML = "<strong>" + play.Track + "</strong> <span class=\"mid-gray\">by</span> " + play.Artist;
		link.className = "no-underline dark-gray";
		track.appendChild(link);
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
