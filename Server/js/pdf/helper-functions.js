const getWindowSize = () => {
    return { x: pdf.container.offsetWidth, y: pdf.container.offsetHeight };
}

const getScaleFactor = (before, after) => {
    x = after.x / before.x;
    y = after.y / before.y;
    return { x: x, y: y };
}

const applyScaleFactor = (cords, scaleFactor) => {
    return { x: cords.x * scaleFactor.x, y: cords.y * scaleFactor.y };
}

const outlineToggle = () => {
    outline = document.getElementById("outline");
    outlineToggler = document.getElementById("outline-toggle");
    if (outline.style.width == "250px") {
        outline.style.width = "0px";
        outline.style.borderRadius = "0 0 0 0";
        outlineToggler.style.marginLeft = "0px";
        outlineToggler.style.borderRadius = "0 0 50px 0";
        outlineToggler.firstChild.src = "/pdf-viewer/icons/hamburgerMenu.svg";
        outlineToggler.firstChild.style.padding = "3px";
    } else {
        outline.style.width = "250px";
        outline.style.borderRadius = "0 20px 0 0";
        outlineToggler.style.marginLeft = "200px";
        outlineToggler.style.borderRadius = "0 20px 0 0";
        outlineToggler.firstChild.src = "/pdf-viewer/icons/backArrow.svg";
        outlineToggler.firstChild.style.padding = "10px";
    }
}

const getCords = () => { return { x: void 0 !== window.pageXOffset ? window.pageXOffset : (document.documentElement || document.body.parentNode || document.body).scrollLeft, y: void 0 !== window.pageYOffset ? window.pageYOffset : (document.documentElement || document.body.parentNode || document.body).scrollTop } },
    scroll = o => { window.scroll({ top: o.y, left: o.x, behavior: "smooth" }) };