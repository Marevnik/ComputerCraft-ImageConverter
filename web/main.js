"use strict";

const UPLOAD_ENDPOINT = "/CCIC/uploadimage";
const GIF_UPLOAD_ENDPOINT = "/CCIC/uploadgif";
const CONTENT_LIST_ENDPOINT = "/getAvaliableContent";
const CONTENT_CHANGE_ENDPOINT = "/ChangeCurrentContent";
const CONTENT_FILE_ENDPOINT = "/CCIC/content/";
const MAX_FILE_SIZE = 25 * 1024 * 1024;
const MIN_CROP_SIZE = 36;

const elements = {
    fileInput: document.querySelector("#fileInput"),
    chooseButton: document.querySelector("#chooseButton"),
    replaceButton: document.querySelector("#replaceButton"),
    resetButton: document.querySelector("#resetButton"),
    dropZone: document.querySelector("#dropZone"),
    emptyState: document.querySelector("#emptyState"),
    cropStage: document.querySelector("#cropStage"),
    canvas: document.querySelector("#cropCanvas"),
    cropSelection: document.querySelector("#cropSelection"),
    cropSize: document.querySelector("#cropSize"),
    cropHelp: document.querySelector("#cropHelp"),
    gifStage: document.querySelector("#gifStage"),
    gifPreview: document.querySelector("#gifPreview"),
    sendButton: document.querySelector("#sendButton"),
    sendButtonText: document.querySelector("#sendButtonText"),
    status: document.querySelector("#status"),
    tabButtons: [...document.querySelectorAll("[data-tab]")],
    tabPanels: [...document.querySelectorAll(".tab-panel")],
    galleryGrid: document.querySelector("#galleryGrid"),
    galleryState: document.querySelector("#galleryState"),
    galleryStatus: document.querySelector("#galleryStatus"),
    galleryCount: document.querySelector("#galleryCount"),
    refreshGalleryButton: document.querySelector("#refreshGalleryButton")
};

const context = elements.canvas.getContext("2d", { alpha: false });
const state = {
    image: null,
    sourceFile: null,
    objectUrl: null,
    stageWidth: 0,
    stageHeight: 0,
    crop: { x: 0, y: 0, width: 0, height: 0 },
    interaction: null,
    isGif: false,
    isSending: false,
    galleryLoaded: false,
    galleryLoading: false,
    selectedContent: ""
};

function clamp(value, min, max) {
    return Math.min(Math.max(value, min), max);
}

function setStatus(message, type = "") {
    elements.status.textContent = message;
    elements.status.className = `status${type ? ` is-${type}` : ""}`;
}

function setEditorVisibility(hasImage) {
    elements.emptyState.hidden = hasImage;
    elements.cropStage.hidden = !hasImage || state.isGif;
    elements.gifStage.hidden = !hasImage || !state.isGif;
    elements.cropHelp.hidden = !hasImage || state.isGif;
    elements.replaceButton.hidden = !hasImage;
    elements.resetButton.hidden = !hasImage || state.isGif;
    elements.sendButton.disabled = !hasImage || state.isSending;
    elements.sendButtonText.textContent = state.isGif ? "Upload GIF" : "Upload image";
    setStatus(
        hasImage
            ? state.isGif ? "The entire GIF will be uploaded" : "Select an area and upload the image"
            : "Choose an image first"
    );
}

function displayLimits() {
    return {
        width: Math.min(900, Math.max(240, elements.dropZone.clientWidth - 52)),
        height: Math.min(500, Math.max(220, elements.dropZone.clientHeight - 52))
    };
}

function layoutImage(preserveCrop = false) {
    if (!state.image) return;

    const previousWidth = state.stageWidth;
    const previousHeight = state.stageHeight;
    const normalized = previousWidth && previousHeight ? {
        x: state.crop.x / previousWidth,
        y: state.crop.y / previousHeight,
        width: state.crop.width / previousWidth,
        height: state.crop.height / previousHeight
    } : null;

    const limits = displayLimits();
    const scale = Math.min(
        limits.width / state.image.naturalWidth,
        limits.height / state.image.naturalHeight
    );
    state.stageWidth = Math.max(1, Math.round(state.image.naturalWidth * scale));
    state.stageHeight = Math.max(1, Math.round(state.image.naturalHeight * scale));

    elements.cropStage.style.width = `${state.stageWidth}px`;
    elements.cropStage.style.height = `${state.stageHeight}px`;

    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    elements.canvas.width = Math.round(state.stageWidth * dpr);
    elements.canvas.height = Math.round(state.stageHeight * dpr);
    context.setTransform(dpr, 0, 0, dpr, 0, 0);
    context.imageSmoothingEnabled = true;
    context.imageSmoothingQuality = "high";
    context.drawImage(state.image, 0, 0, state.stageWidth, state.stageHeight);

    if (preserveCrop && normalized) {
        state.crop = {
            x: normalized.x * state.stageWidth,
            y: normalized.y * state.stageHeight,
            width: normalized.width * state.stageWidth,
            height: normalized.height * state.stageHeight
        };
        normalizeCrop();
        renderCrop();
    } else {
        resetCrop();
    }
}

function resetCrop() {
    const insetX = state.stageWidth * 0.1;
    const insetY = state.stageHeight * 0.1;
    state.crop = {
        x: insetX,
        y: insetY,
        width: state.stageWidth - insetX * 2,
        height: state.stageHeight - insetY * 2
    };
    normalizeCrop();
    renderCrop();
}

function normalizeCrop() {
    const minimumWidth = Math.min(MIN_CROP_SIZE, state.stageWidth);
    const minimumHeight = Math.min(MIN_CROP_SIZE, state.stageHeight);
    state.crop.width = clamp(state.crop.width, minimumWidth, state.stageWidth);
    state.crop.height = clamp(state.crop.height, minimumHeight, state.stageHeight);
    state.crop.x = clamp(state.crop.x, 0, state.stageWidth - state.crop.width);
    state.crop.y = clamp(state.crop.y, 0, state.stageHeight - state.crop.height);
}

function sourceCrop() {
    const scaleX = state.image.naturalWidth / state.stageWidth;
    const scaleY = state.image.naturalHeight / state.stageHeight;
    return {
        x: Math.round(state.crop.x * scaleX),
        y: Math.round(state.crop.y * scaleY),
        width: Math.max(1, Math.round(state.crop.width * scaleX)),
        height: Math.max(1, Math.round(state.crop.height * scaleY))
    };
}

function renderCrop() {
    const crop = state.crop;
    elements.cropSelection.style.transform = `translate(${crop.x}px, ${crop.y}px)`;
    elements.cropSelection.style.width = `${crop.width}px`;
    elements.cropSelection.style.height = `${crop.height}px`;
    if (state.image) {
        const source = sourceCrop();
        elements.cropSize.textContent = `${source.width} × ${source.height} px`;
    }
}

function pointInStage(event) {
    const bounds = elements.cropStage.getBoundingClientRect();
    return {
        x: clamp(event.clientX - bounds.left, 0, state.stageWidth),
        y: clamp(event.clientY - bounds.top, 0, state.stageHeight)
    };
}

function beginInteraction(event, mode, direction = "") {
    const point = pointInStage(event);
    state.interaction = {
        pointerId: event.pointerId,
        mode,
        direction,
        startX: point.x,
        startY: point.y,
        crop: { ...state.crop }
    };
    elements.cropStage.setPointerCapture(event.pointerId);
    elements.cropStage.classList.add("is-interacting");
}

function moveSelection(dx, dy, start) {
    state.crop.x = clamp(start.x + dx, 0, state.stageWidth - start.width);
    state.crop.y = clamp(start.y + dy, 0, state.stageHeight - start.height);
}

function resizeSelection(dx, dy, start, direction) {
    const right = start.x + start.width;
    const bottom = start.y + start.height;
    let x = start.x;
    let y = start.y;
    let width = start.width;
    let height = start.height;

    if (direction.includes("e")) width = clamp(start.width + dx, MIN_CROP_SIZE, state.stageWidth - start.x);
    if (direction.includes("s")) height = clamp(start.height + dy, MIN_CROP_SIZE, state.stageHeight - start.y);
    if (direction.includes("w")) {
        x = clamp(start.x + dx, 0, right - MIN_CROP_SIZE);
        width = right - x;
    }
    if (direction.includes("n")) {
        y = clamp(start.y + dy, 0, bottom - MIN_CROP_SIZE);
        height = bottom - y;
    }

    state.crop = { x, y, width, height };
}

function updateInteraction(event) {
    const interaction = state.interaction;
    if (!interaction || event.pointerId !== interaction.pointerId) return;
    const point = pointInStage(event);
    const dx = point.x - interaction.startX;
    const dy = point.y - interaction.startY;

    if (interaction.mode === "move") {
        moveSelection(dx, dy, interaction.crop);
    } else if (interaction.mode === "resize") {
        resizeSelection(dx, dy, interaction.crop, interaction.direction);
    } else if (interaction.mode === "create") {
        const x = Math.min(interaction.startX, point.x);
        const y = Math.min(interaction.startY, point.y);
        state.crop = {
            x,
            y,
            width: Math.max(MIN_CROP_SIZE, Math.abs(point.x - interaction.startX)),
            height: Math.max(MIN_CROP_SIZE, Math.abs(point.y - interaction.startY))
        };
        normalizeCrop();
    }
    renderCrop();
}

function endInteraction(event) {
    if (!state.interaction || event.pointerId !== state.interaction.pointerId) return;
    state.interaction = null;
    elements.cropStage.classList.remove("is-interacting");
}

function loadFile(file) {
    if (!file) return;
    if (!/^image\/(png|jpeg|gif)$/.test(file.type)) {
        setStatus("Only PNG, JPEG, and GIF files are supported", "error");
        return;
    }
    if (file.size > MAX_FILE_SIZE) {
        setStatus("The file is too large — the maximum size is 25 MB", "error");
        return;
    }

    if (state.objectUrl) URL.revokeObjectURL(state.objectUrl);
    state.objectUrl = URL.createObjectURL(file);
    state.sourceFile = file;
    state.isGif = file.type === "image/gif";

    if (state.isGif) {
        state.image = null;
        elements.gifPreview.src = state.objectUrl;
        setEditorVisibility(true);
        return;
    }

    elements.gifPreview.removeAttribute("src");
    const image = new Image();
    image.onload = () => {
        state.image = image;
        setEditorVisibility(true);
        requestAnimationFrame(() => layoutImage(false));
    };
    image.onerror = () => setStatus("Could not read the image", "error");
    image.src = state.objectUrl;
}

function createCroppedBlob() {
    return new Promise((resolve, reject) => {
        const crop = sourceCrop();
        const output = document.createElement("canvas");
        const maxSide = 4096;
        const reduction = Math.min(1, maxSide / Math.max(crop.width, crop.height));
        output.width = Math.max(1, Math.round(crop.width * reduction));
        output.height = Math.max(1, Math.round(crop.height * reduction));

        const outputContext = output.getContext("2d", { alpha: false });
        outputContext.fillStyle = "#000";
        outputContext.fillRect(0, 0, output.width, output.height);
        outputContext.imageSmoothingEnabled = true;
        outputContext.imageSmoothingQuality = "high";
        outputContext.drawImage(state.image, crop.x, crop.y, crop.width, crop.height, 0, 0, output.width, output.height);
        output.toBlob(blob => blob ? resolve(blob) : reject(new Error("Canvas export failed")), "image/png");
    });
}

async function sendImage() {
    if (!state.sourceFile || state.isSending || (!state.isGif && !state.image)) return;
    state.isSending = true;
    elements.sendButton.disabled = true;
    elements.sendButton.classList.add("is-loading");
    elements.sendButtonText.textContent = "Preparing…";
    setStatus("Preparing the cropped image");

    try {
        const blob = state.isGif ? state.sourceFile : await createCroppedBlob();
        const data = new FormData();
        const baseName = (state.sourceFile?.name || "image").replace(/\.[^.]+$/, "");
        const uploadName = state.isGif ? state.sourceFile.name : `${baseName}-cropped.png`;
        const endpoint = state.isGif ? GIF_UPLOAD_ENDPOINT : UPLOAD_ENDPOINT;
        data.append("image", blob, uploadName);

        elements.sendButtonText.textContent = "Uploading…";
        setStatus(state.isGif ? "Uploading and processing the GIF" : "Uploading the image to the server");
        const response = await fetch(endpoint, { method: "POST", body: data });
        if (!response.ok) {
            const message = (await response.text()).trim();
            throw new Error(message || `The server responded with status ${response.status}`);
        }

        elements.sendButtonText.textContent = "Uploaded";
        setStatus(state.isGif ? "The GIF was processed and uploaded successfully" : "The image was uploaded successfully", "success");
        state.galleryLoaded = false;
    } catch (error) {
        elements.sendButtonText.textContent = "Try again";
        setStatus(error.message || "Could not upload the image", "error");
    } finally {
        state.isSending = false;
        elements.sendButton.disabled = false;
        elements.sendButton.classList.remove("is-loading");
    }
}

function setGalleryState(title, description = "", type = "loading") {
    elements.galleryState.hidden = false;
    elements.galleryState.className = `gallery-state gallery-state--${type}`;
    elements.galleryState.replaceChildren();

    if (type === "loading") {
        const loader = document.createElement("span");
        loader.className = "gallery-loader";
        loader.setAttribute("aria-hidden", "true");
        elements.galleryState.append(loader);
    }

    const heading = document.createElement("strong");
    heading.textContent = title;
    const copy = document.createElement("small");
    copy.textContent = description;
    elements.galleryState.append(heading, copy);
}

function updateSelectedCard() {
    for (const card of elements.galleryGrid.querySelectorAll(".gallery-card")) {
        const selected = card.dataset.filename === state.selectedContent;
        card.classList.toggle("is-selected", selected);
        card.setAttribute("aria-pressed", String(selected));
        const label = card.querySelector(".gallery-card-action");
        if (label) label.textContent = selected ? "Selected" : "Select";
    }
}

async function selectGalleryImage(filename) {
    if (!filename) return;
    elements.galleryStatus.className = "gallery-status";
    elements.galleryStatus.textContent = `Selecting ${filename}…`;

    try {
        const response = await fetch(CONTENT_CHANGE_ENDPOINT, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ name: filename, anim: filename.toLowerCase().endsWith(".gif.png") })
        });
        if (!response.ok) {
            const message = (await response.text()).trim();
            throw new Error(message || `The server responded with status ${response.status}`);
        }

        state.selectedContent = filename;
        updateSelectedCard();
        elements.galleryStatus.className = "gallery-status is-success";
        elements.galleryStatus.textContent = `${filename} is now selected for display`;
    } catch (error) {
        elements.galleryStatus.className = "gallery-status is-error";
        elements.galleryStatus.textContent = error.message || "Could not select the image";
    }
}

function createGalleryCard(filename) {
    const card = document.createElement("button");
    card.type = "button";
    card.className = "gallery-card";
    card.dataset.filename = filename;
    card.setAttribute("aria-pressed", "false");
    card.setAttribute("aria-label", `Select image ${filename}`);

    const preview = document.createElement("span");
    preview.className = "gallery-card-preview";
    const image = document.createElement("img");
    image.src = `${CONTENT_FILE_ENDPOINT}${encodeURIComponent(filename)}`;
    image.alt = "";
    image.loading = "lazy";
    image.addEventListener("error", () => preview.classList.add("has-error"));
    preview.append(image);

    if (filename.toLowerCase().endsWith(".gif.png")) {
        const badge = document.createElement("span");
        badge.className = "gallery-animation-badge";
        badge.textContent = "GIF";
        preview.append(badge);
    }

    const meta = document.createElement("span");
    meta.className = "gallery-card-meta";
    const name = document.createElement("span");
    name.className = "gallery-card-name";
    name.textContent = filename;
    name.title = filename;
    const action = document.createElement("span");
    action.className = "gallery-card-action";
    action.textContent = "Select";
    meta.append(name, action);
    card.append(preview, meta);
    card.addEventListener("click", () => selectGalleryImage(filename));
    return card;
}

async function loadGallery(force = false) {
    if (state.galleryLoading || (state.galleryLoaded && !force)) return;
    state.galleryLoading = true;
    elements.refreshGalleryButton.disabled = true;
    elements.galleryGrid.replaceChildren();
    elements.galleryStatus.textContent = "";
    setGalleryState("Loading images", "Fetching available content");

    try {
        const response = await fetch(CONTENT_LIST_ENDPOINT, { method: "GET", cache: "no-store" });
        if (!response.ok) throw new Error(`The server responded with status ${response.status}`);
        const data = await response.json();
        if (!Array.isArray(data)) throw new Error("The server returned an invalid list format");

        const filenames = data
            .filter(name => typeof name === "string" && name.toLowerCase().endsWith(".png"))
            .sort((a, b) => a.localeCompare(b, "ru", { numeric: true }));

        elements.galleryCount.textContent = String(filenames.length);
        elements.galleryCount.hidden = false;
        state.galleryLoaded = true;

        if (!filenames.length) {
            setGalleryState("The gallery is empty", "Upload your first image from the Upload tab", "empty");
            return;
        }

        elements.galleryState.hidden = true;
        const fragment = document.createDocumentFragment();
        for (const filename of filenames) fragment.append(createGalleryCard(filename));
        elements.galleryGrid.append(fragment);
        updateSelectedCard();
    } catch (error) {
        setGalleryState("Could not load the gallery", error.message || "Check your connection to the server", "error");
    } finally {
        state.galleryLoading = false;
        elements.refreshGalleryButton.disabled = false;
    }
}

function activateTab(panelId) {
    for (const button of elements.tabButtons) {
        const active = button.dataset.tab === panelId;
        button.classList.toggle("is-active", active);
        button.setAttribute("aria-selected", String(active));
        button.tabIndex = active ? 0 : -1;
    }
    for (const panel of elements.tabPanels) {
        const active = panel.id === panelId;
        panel.hidden = !active;
        panel.classList.toggle("is-active", active);
    }
    if (panelId === "galleryPanel") loadGallery();
}

function openFilePicker() {
    elements.fileInput.value = "";
    elements.fileInput.click();
}

elements.chooseButton.addEventListener("click", openFilePicker);
elements.replaceButton.addEventListener("click", openFilePicker);
elements.fileInput.addEventListener("change", event => loadFile(event.target.files[0]));
elements.resetButton.addEventListener("click", resetCrop);
elements.sendButton.addEventListener("click", sendImage);
elements.refreshGalleryButton.addEventListener("click", () => loadGallery(true));
for (const button of elements.tabButtons) {
    button.addEventListener("click", () => activateTab(button.dataset.tab));
}

for (const eventName of ["dragenter", "dragover"]) {
    elements.dropZone.addEventListener(eventName, event => {
        event.preventDefault();
        elements.dropZone.classList.add("is-dragging");
    });
}
for (const eventName of ["dragleave", "drop"]) {
    elements.dropZone.addEventListener(eventName, event => {
        event.preventDefault();
        elements.dropZone.classList.remove("is-dragging");
    });
}
elements.dropZone.addEventListener("drop", event => loadFile(event.dataTransfer.files[0]));

elements.cropSelection.addEventListener("pointerdown", event => {
    event.preventDefault();
    event.stopPropagation();
    const handle = event.target.closest("[data-direction]");
    beginInteraction(event, handle ? "resize" : "move", handle?.dataset.direction || "");
});

elements.canvas.addEventListener("pointerdown", event => {
    event.preventDefault();
    beginInteraction(event, "create");
    const point = pointInStage(event);
    state.crop = { x: point.x, y: point.y, width: MIN_CROP_SIZE, height: MIN_CROP_SIZE };
    normalizeCrop();
    renderCrop();
});

elements.cropStage.addEventListener("pointermove", updateInteraction);
elements.cropStage.addEventListener("pointerup", endInteraction);
elements.cropStage.addEventListener("pointercancel", endInteraction);

let resizeFrame = 0;
window.addEventListener("resize", () => {
    if (!state.image || state.isGif) return;
    cancelAnimationFrame(resizeFrame);
    resizeFrame = requestAnimationFrame(() => layoutImage(true));
});
window.addEventListener("beforeunload", () => {
    if (state.objectUrl) URL.revokeObjectURL(state.objectUrl);
});

setEditorVisibility(false);
