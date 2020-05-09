const url = './sample.pdf';


const container = document.getElementById('container');
const currentPage = document.getElementById('currentPage');
const totalPages = document.getElementById('totalPages');

const canvas = document.querySelector('#pdf-render'),
  ctx = canvas.getContext('2d');

// Render the page
const renderPage = num => {
    PDF.pageIsRendering = true;

  // Get page
  PDF.doc.getPage(num).then(page => {
    // Set scale
    let scale = PDF.scale;
    const viewport = page.getViewport({ scale });
    canvas.height = viewport.height;
    canvas.width = viewport.width;

    const renderCtx = {
      canvasContext: ctx,
      viewport
    };
    page.render(renderCtx).promise.then(() => {
        PDF.pageIsRendering = false;
        window.scroll({ top: 0, left: 0, behavior: "smooth" })
        if (PDF.pageNumIsPending !== null) {
            renderPage(PDF.pageNumIsPending);
            PDF.pageNumIsPending = null;
        }
        window.scroll({ top: 0, left: 0, behavior: "smooth" })
        updatePdfProgress()
    });
  });
};

// Check for pages rendering
const queueRenderPage = num => {
  if (PDF.pageIsRendering) {
    PDF.pageNumIsPending = num;
  } else {
    renderPage(num);
  }
};

// Show Prev Page
const showPrevPage = () => {
  if (PDF.pageNum <= 1) {
    return;
  }
  PDF.pageNum--;
  currentPage.innerText = PDF.pageNum;
  queueRenderPage(PDF.pageNum);
};

// Show Next Page
const showNextPage = () => {
  if (PDF.pageNum >= PDF.doc.numPages) {
    return;
  }
  PDF.pageNum++;
  currentPage.innerText = PDF.pageNum;
  queueRenderPage(PDF.pageNum);
};

// Get Document
pdfjsLib
  .getDocument(PDF.url)
  .promise.then(pdfDoc_ => {
    PDF.doc = pdfDoc_;
    totalPages.innerText = " "+PDF.doc.numPages;
    currentPage.innerText = PDF.pageNum+" ";
    renderPage(PDF.pageNum);
  })
  .catch(err => {
    // Display error
    const div = document.createElement('div');
    div.className = 'error';
    div.appendChild(document.createTextNode(err.message));
    document.querySelector('.container').insertBefore(div, canvas);
  });

  document.onkeydown = function(e) {
    if (e.keyCode == 37){
        showPrevPage();
    } else if (e.keyCode == 39){
        showNextPage();
    } else if (e.keyCode == 27){
        res="";
        path = PDF.path.split("/")
        for(i=1; i < path.length-1;i++)
          res +="/"+path[i]
        window.location.href = "/home?path=" + res;
    }
}

const updatePdfProgress = ()=> {
    let body = {
        "Result": null,
        "VariableType": '{"Hash": "TEXT","Username": "TEXT","Page":"INTEGER"}',
        "Contains": '{"Hash": "'+PDF.hash+'","Username": "'+PDF.user+'"}',
        "Set": '{"Page":"'+PDF.pageNum+'"}',
        "TableName": "PDFS",
        "DatabaseOperation": "UPDATE"
      }
    post("/query", body)
}
