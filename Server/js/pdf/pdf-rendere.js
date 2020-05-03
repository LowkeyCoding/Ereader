// Loaded via <script> tag, create shortcut to access PDF.js exports.
var pdfjsLib = window['pdfjs-dist/build/pdf'];
// The workerSrc property shall be specified.
pdfjsLib.GlobalWorkerOptions.workerSrc = 'js/libs/pdf.worker.min.js';

var pdf = {
    document: null,
    documentID: null,
    pageNumber: 1,
    pagesRendered: 4,
    scale: 0.6,
    container: null,
};

// Render the page
const setupPage = (page) => {
    var canvas = document.createElement('canvas');
    var context = canvas.getContext('2d');
    // Set scale
    const viewport = calculateViewport(page)
    canvas.height = viewport.height;
    canvas.width = viewport.width;
    const renderContext = {
        canvasContext: context,
        viewport
    };
    return { canvas, renderContext }
};

const renderPages = () => {
    for (i = pdf.pageNumber; i <= (pdf.pageNumber + pdf.pagesRendered) - 1; i++) {
        pdf.document.getPage(i).then((page) => {
            pageConfig = setupPage(page);
            pdf.viewport = pageConfig.renderContext.viewport;
            pdf.container.appendChild(pageConfig.canvas);
            page.render(pageConfig.renderContext);
        });
    }
}

// Chaning pages
const nextPage = async() => {
    //get current progress
    progress = await getProgress();
    //Replace the old pages
    pdf.pageNumber++;
    await replacePages();
    //Scroll to the top of the last rendered page
    newPageLength = pdf.pages[pdf.pagesRendered - 1].offsetHeight;
    oldPageLength = pdf.pages[pdf.pagesRendered - 2].offsetHeight;
    oldPageProgress = (newPageLength * progress.currentPageProgress);
    scrollLocation = pdf.container.offsetHeight - (newPageLength + oldPageLength - oldPageProgress);
    window.scrollTo({
        top: scrollLocation,
        left: 0,
        behavior: 'auto'
    });
}

const previousPage = async() => {
    // Replace the old pages
    pdf.pageNumber--;
    await replacePages();
    // Scroll to the bottom of page 0
    window.scrollTo({
        top: pdf.pages[0].offsetHeight,
        left: 0,
        behavior: 'auto'
    });
}

const replacePages = async() => {
    for (i = pdf.pageNumber; i <= (pdf.pageNumber + pdf.pagesRendered) - 1; i++) {
        if (i <= pdf.document.numPages) {
            await pdf.document.getPage(i).then((page) => {
                pageConfig = setupPage(page); // get page config
                pdf.container.replaceChild(pageConfig.canvas, pdf.pages[(i - pdf.pageNumber)]); // Replace the canvas
                page.render(pageConfig.renderContext); // Render the page
            }).catch((error) => {
                console.log(error)
            });
        }
    }
}

const gotoPage = async(pageNumber) => {
    pdf.pageNumber = pageNumber; // Change page number
    await replacePages(); // Rerender pages
}

const gotoOutlineItem = async(dest) => {
    // Get destination info
    destination = await pdf.document.getDestination(dest.toString());
    // Get page number
    pageNumber = await pdf.document.getPageIndex(destination[0]) + 1;
    //Navigate to page
    await gotoPage(pageNumber);
    // The cordinates are off by around +-200 with a reselution of 1920x1080
    window.scrollTo({
        top: destination[3],
        left: destination[2],
        behavior: 'auto'
    });
}

// Generate Outline
const generateOutline = async(outlineContainer, outline) => {
    for (i in outline) {
        if (outline[i].items.length <= 0) {
            element = document.createElement("outlineItem");
            element.dest = outline[i].dest;
            element.id = outline[i].title;
            element.innerText = element.id;
            element.onclick = async() => {
                await gotoOutlineItem(event.target.dest);
            }
            outlineContainer.appendChild(element);
        } else {
            element = document.createElement("subOutline");
            element.id = outline[i].title;
            element.innerText = element.id;
            element.dest = outline[i].dest;
            element.onclick = async(event) => {
                await gotoOutlineItem(event.target.dest);
            }
            element = await generateOutline(element, outline[i].items);
            outlineContainer.appendChild(element);
        }
    }
    return outlineContainer;
}

// Helper functions
const calculateViewport = (page) => {
    let clientWidth = Math.min(document.documentElement.clientWidth, window.innerWidth || 0);
    let viewport = page.getViewport({ scale: clientWidth / page.getViewport({ scale: 1 }).width });
    return viewport;
}

const init = async(url, element) => {
    pdf.container = element;
    pdfjsLib
        .getDocument(url, {cache: "force-cache"})
        .promise.then(async(_pdf) => {
            pdf.scale = 0.99;
            pdf.document = _pdf;
            pdf.documentID = pdf.document.fingerprint;
            pdf.pageNumber = config.page
            await renderPages();
            pdf.pages = pdf.container.childNodes;
            outlineContainer = document.getElementById("outline");
            outline = await pdf.document.getOutline();
            await generateOutline(outlineContainer, outline);
        }).then(
            window.scrollTo({
                top: config.cords.y,
                left: config.cords.x,
                behavior: 'auto'
            })
        ).catch(err => { console.error(err) });
}

const getProgress = async() => {
    y = getCords().y
    currentPage = pdf.pageNumber;
    currentPageProgress = 0;
    for (i = 0; i <= pdf.pages.length - 1; i++) {
        height = pdf.pages[i].offsetHeight;
        y -= height;
        if (y < 0) {
            currentPageProgress = (y + height) / height;
            break;
        } else {
            currentPage++;
        }
    }
    return { documentID: pdf.document._pdfInfo.fingerprint, currentPage, currentPageProgress }
}