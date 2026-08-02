local monitor = assert(peripheral.find("monitor"))
monitor.setTextScale(0.5)

local WIDTH, HEIGHT = monitor.getSize()

local SERVER = "http://localhost:4243/"

local FPS = 20
local POLL_EVERY = 6 -- проверять сервер каждые 6 кадров

--------------------------------------------------
-- Вспомогательная функция POST JSON
--------------------------------------------------

local function postJSON(path, data)
    local body = textutils.serializeJSON(data)

    return http.post(
        SERVER .. path,
        body,
        {
            ["Content-Type"] = "application/json"
        }
    )
end

--------------------------------------------------
-- Отправка размеров монитора
--------------------------------------------------

local function sendMonitorSize()
    local response, err = postJSON("/setSize", {
        width = WIDTH,
        height = HEIGHT
    })

    if not response then
        print("Failed to send monitor size:", err)
        return false
    end

    print("Size response:", response.getResponseCode())
    response.close()

    return true
end

--------------------------------------------------
-- Загрузка палитры
--------------------------------------------------

local function loadPalette(path)
    local file = assert(io.open(path, "r"))

    local palette = {}
    local i = 0

    for line in file:lines() do
        line = line:match("^%s*(.-)%s*$")

        palette[i] = assert(
            tonumber(line, 16),
            "incorrect color: " .. line
        )

        i = i + 1
    end

    file:close()

    assert(
        i == 16,
        "expected 16 colors, got: " .. i
    )

    return palette
end

--------------------------------------------------
-- Загрузка кадров
--------------------------------------------------

local function loadFrames(path)
    local file = assert(io.open(path, "r"))

    local frames = {}
    local frame = {}
    local y = 1

    for line in file:lines() do
        frame[y] = line
        y = y + 1

        if y > HEIGHT then
            frames[#frames + 1] = frame

            frame = {}
            y = 1
        end
    end

    file:close()

    assert(
        y == 1,
        "pixels file contains incomplete frame"
    )

    return frames
end

--------------------------------------------------
-- Отрисовка
--------------------------------------------------

local bg = string.rep("0", WIDTH)
local text = string.rep(" ", WIDTH)

local function drawFrame(frame)
    for y = 1, HEIGHT do
        monitor.setCursorPos(1, y)
        monitor.blit(text, bg, frame[y])
    end
end

--------------------------------------------------
-- Текущее состояние
--------------------------------------------------

local currentContent = {
    name = "none",
    anim = false
}

local palette = nil
local frames = nil

--------------------------------------------------
-- Удаление .png только для пути
--------------------------------------------------

local function getContentBaseName(name)
    return name:gsub("%.png$", "")
end

--------------------------------------------------
-- Загрузка выбранного контента
--------------------------------------------------

local function loadContent(content)
    if content.name == "none" then
        print("No content selected")
        currentContent = content
        return false
    end

    print("Loading:", content.name)

    -- cat.png -> cat
    local baseName = getContentBaseName(content.name)

    local basePath = "/monitor/imgs/" .. baseName

    local palettePath = basePath .. "palette.txt"
    local pixelsPath = basePath .. "pixels.txt"

    print("Palette:", palettePath)
    print("Pixels:", pixelsPath)

    -- Сначала полностью загружаем новое изображение.
    -- Старые данные пока остаются рабочими.
    local newPalette = loadPalette(palettePath)
    local newFrames = loadFrames(pixelsPath)

    -- Только после успешной загрузки заменяем данные.
    palette = newPalette
    frames = newFrames

    for i = 0, 15 do
        monitor.setPaletteColor(
            2 ^ i,
            palette[i]
        )
    end

    -- ВАЖНО:
    -- сохраняем именно оригинальный content,
    -- то есть name всё ещё будет "cat.png".
    currentContent = content

    print("Loaded:", baseName)
    print("Frames:", #frames)

    return true
end

--------------------------------------------------
-- Инициализация состояния на сервере
--------------------------------------------------

local function initContent()
    local response, err = postJSON(
        "/CCInitContent",
        currentContent
    )

    if not response then
        print("Failed to initialize content:", err)
        return false
    end

    local code = response.getResponseCode()

    if code == 202 then
        print("Server content initialized")
    elseif code == 409 then
        -- Сервер уже был инициализирован.
        -- Это не фатальная ошибка.
        print("Server content already initialized")
    else
        print("Unexpected init response:", code)
    end

    response.close()

    return code == 202 or code == 409
end

--------------------------------------------------
-- Проверка обновления
--------------------------------------------------

local function checkForUpdate()
    local response, err = postJSON(
        "/CCMonitorGetUpdateStatus",
        currentContent
    )

    if not response then
        print("Update check failed:", err)
        return false
    end

    local code = response.getResponseCode()

    --------------------------------------------------
    -- 204 = серверное состояние совпадает с нашим
    --------------------------------------------------

    if code == 204 then
        response.close()
        return false
    end

    --------------------------------------------------
    -- Всё кроме 200/204 считаем ошибкой
    --------------------------------------------------

    if code ~= 200 then
        print("Unexpected update response:", code)
        response.close()
        return false
    end

    --------------------------------------------------
    -- Получили новый CurrentContent
    --------------------------------------------------

    local body = response.readAll()
    response.close()

    local newContent = textutils.unserializeJSON(body)

    if not newContent then
        print("Invalid JSON from server")
        return false
    end

    if not newContent.name then
        print("Server response has no content name")
        return false
    end

    print("New content:", newContent.name)
    print("Anim:", tostring(newContent.anim))

    return loadContent(newContent)
end

--------------------------------------------------
-- Запуск
--------------------------------------------------

sendMonitorSize()
initContent()

-- На случай, если сервер уже содержит другой контент.
checkForUpdate()

--------------------------------------------------
-- Главный цикл
--------------------------------------------------

-- Счётчик должен жить за пределами цикла кадров.
-- У статичного изображения #frames == 1, поэтому проверка через
-- номер кадра i никогда не доходила до POLL_EVERY.
local framesSincePoll = 0

while true do
    if frames and #frames > 0 then

        for i = 1, #frames do
            drawFrame(frames[i])

            framesSincePoll = framesSincePoll + 1

            --------------------------------------------------
            -- Периодически спрашиваем сервер
            --------------------------------------------------

            if framesSincePoll >= POLL_EVERY then
                framesSincePoll = 0
                local updated = checkForUpdate()

                if updated then
                    -- frames уже указывает на новый контент.
                    -- Выходим из воспроизведения старого.
                    break
                end
            end

            sleep(1 / FPS)
        end

    else
        --------------------------------------------------
        -- Если ничего ещё не выбрано,
        -- ждём выбора с сайта.
        --------------------------------------------------

        checkForUpdate()
        sleep(0.5)
    end
end
