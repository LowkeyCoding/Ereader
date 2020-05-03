// Resize the pdf when the window is resized
window.addEventListener('resize', async() => {
    cords = getCords();
    if (!document.beforeWindowSize) {
        document.beforeWindowSize = getWindowSize();
    }
    while (pdf.container.firstChild) {
        pdf.container.removeChild(pdf.container.lastChild);
    }
    await renderPages(); // We need to wait for the pages to render before chaning the window scroll
    windowSize = getWindowSize();
    if (JSON.stringify(document.beforeWindowSize) != JSON.stringify(windowSize)) {
        scaleFactor = getScaleFactor(document.beforeWindowSize, windowSize);
        cords = applyScaleFactor(cords, scaleFactor);
        document.beforeWindowSize = windowSize;
    }
    window.scrollTo(cords.x, cords.y);
});

// Save the current page progress across platforms & render new pages when at bottom and top of container
var isScrolling;

window.addEventListener('scroll', async(event) => {
    event.stopPropagation();
    window.clearTimeout(isScrolling);
    isScrolling = setTimeout(async() => {
        progress = await getProgress();
        page = (progress.currentPage - pdf.pageNumber) + 1
        if (page <= pdf.document.numPages) {
            if (page == pdf.pagesRendered) {
                await nextPage();
            } else if (page == 1 && progress.currentPage != 1 && progress.currentPageProgress == 0) {
                await previousPage();
            }
        }
    }, 66);
}, false);

// Create event listener that listens for key presses
window.addEventListener('keypress', async(event) => {
    if (event.key == 'G' && event.shiftKey == true) {
        // Create popup that accepts a page number then go to page
        pageNumber = prompt("Min: 0, Max: " + pdf.document.numPages + "\n Go to page: ");
        if (pageNumber <= pdf.document.numPages) {
            await gotoPage(parseInt(pageNumber));
        }
    }
});