

var canvas = document.getElementById("myCanvas");
var ctx = canvas.getContext("2d");

var myVar = setInterval(myTimer, 500);

sampleStats =  {"con":0,"sub":0,"buf":0,"bps":0.1,"name":"aide0","http":"","tcp":"","Limits":{"con":16,"bps":10,"sub":64}}

function getStats( count ) {

    val = count * 0.1
    if (val > 1 ) 
        val = 1;

    stats = sampleStats
    stats.con = val
    stats.sub = val
    stats.buf = val
    stats.bps = val

    return stats
}

var nodeCount = 0;

function makeGuru( ) {

    guru = {}
    guru.stats = sampleStats
    guru.limits = sampleStats.Limits
    guru.stats.name = "guru" + nodeCount
    nodeCount ++ 
    
    return guru

}

function makeAide( ) {

    aide = makeGuru()
    aide.stats.name = "guru" + nodeCount
    nodeCount ++ 
    
    return aide

}

var count = 0

// @ts-check
function drawExecutive( ctx, stats) {

    ctx.fillStyle = "#111111";
    ctx.strokeRect(0, 0, 120, 80); // surrounding

    barstart = 30
    barend = 90
    var grd = ctx.createLinearGradient(barstart, 0, barend, 0);
    grd.addColorStop(0, "green");
    grd.addColorStop(1, "#FF0000");

    ctx.font = "11px Ariel";
    ctx.codefont = "11px Monaco";

    ctx.fillStyle = grd;
    yoff = 0
    for (const key in stats) {
        if (key == "http" || key == "tpc" ||key == "Limits" )
            break;
        ctx.fillStyle = "#111111";
        val = stats[key]
        ctx.fillText(key, 4, yoff + 10);
        if (key == 'name') {
            ctx.codefont = "11px Monaco";
            ctx.fillText(val, 30, yoff + 10);
            ctx.font = "11px Ariel";
        } else {
            limitval = stats.Limits[key]
            if ( limitval == null)
                limitval = 1.0
            limitstr = getLimitString(limitval)

            ctx.fillText(limitstr, 13 + barend, yoff + 10);

            ctx.fillStyle = grd
            wid = barend - barstart
            wid  = wid * val
            ctx.fillRect(barstart,yoff+4,barstart + wid,8)
            ctx.fillStyle = "#111111";
         }
        yoff += 12
    }
};

function getLimitString( val ) {
    e = 0
    while (val >= 10) {
        e ++
        val /= 10
    }
    if ( e == 0 )
        return ""+val
    return "" + Math.round(val) + "e" + e
}

g1 = makeGuru()
a1 = makeAide()

function myTimer() {

    console.log("hello" + count)
    count ++
    if (count > 400) {
        count = 0
    }

    ctx.fillStyle = "#EEEEEE";
    ctx.fillRect(0, 0, 1512, 1512);

 
    ctx.save();
    ctx.translate(10, 10);
    drawExecutive(ctx,  g1.stats)
    ctx.restore();

    ctx.save();
    ctx.translate(140, 10);
    drawExecutive(ctx, a1.stats)
    ctx.restore();

 
  
}

myTimer()
var myVar = setInterval(myTimer, 2000);


