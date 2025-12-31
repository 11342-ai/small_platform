const { createApp, ref, onMounted, nextTick } = Vue;

createApp({
    setup() {
        const models = ref([]);
        const sessions = ref([]);
        const selectedModel = ref('');
        const sessionId = ref('');
        const messages = ref([]);
        const inputMessage = ref('');
        const isTyping = ref(false);
        const chatContainer = ref(null);
        const personas = ref([]);
        const selectedPersona = ref('')
        const uploadedFiles = ref([]);
        const fileInput = ref(null);

        const API_BASE = '/api';

        // 配置 Marked 和 Highlight.js
        const configureMarkdown = () => {
            // 配置 marked 选项
            marked.setOptions({
                highlight: function(code, lang) {
                    const language = hljs.getLanguage(lang) ? lang : 'plaintext';
                    return hljs.highlight(code, { language }).value;
                },
                langPrefix: 'hljs language-',
                pedantic: false,
                gfm: true,
                breaks: true,
                sanitize: false,
                smartLists: true,
                smartypants: false,
                xhtml: false
            });

            // 配置 highlight.js
            hljs.configure({
                tabReplace: '    ', // 4 spaces
                classPrefix: 'hljs-',
                languages: ['javascript', 'python', 'java', 'go', 'c', 'cpp', 'html', 'css', 'json', 'xml', 'bash', 'sql', 'markdown']
            });
        };

        // Markdown 渲染函数
        const renderMarkdown = (content) => {
            try {
                return marked.parse(content);
            } catch (error) {
                console.error('Markdown 渲染错误:', error);
                return content; // 如果渲染失败，返回原始内容
            }
        };

        const triggerFileInput = () => {
            console.log('触发文件选择'); // 调试日志
            if (fileInput.value) {
                fileInput.value.click();
            } else {
                console.error('fileInput 未找到');
            }
        };

        const handleFileUpload = (event) => {
            console.log('文件选择变化', event.target.files); // 调试日志
            const files = Array.from(event.target.files);
            files.forEach(file => {
                // 检查文件类型
                const allowedTypes = ['.txt', '.py', '.go', '.c', '.cpp', '.h', '.hpp', '.js', '.ts', '.java', '.html', '.css', '.md', '.json', '.xml', '.yaml', '.yml'];
                const fileExt = '.' + file.name.split('.').pop().toLowerCase();

                if (!allowedTypes.includes(fileExt)) {
                    alert(`不支持的文件类型: ${fileExt}`);
                    return;
                }

                // 检查文件大小（限制为5MB）
                if (file.size > 5 * 1024 * 1024) {
                    alert(`文件太大: ${file.name}，请选择小于5MB的文件`);
                    return;
                }

                uploadedFiles.value.push({
                    id: Date.now() + Math.random(),
                    file: file,
                    name: file.name,
                    size: file.size,
                    type: file.type
                });
            });

            // 清空input以便选择相同文件
            event.target.value = '';
        };

        const removeFile = (fileToRemove) => {
            uploadedFiles.value = uploadedFiles.value.filter(file => file.id !== fileToRemove.id);
        };

        const uploadFiles = async (sessionID) => {
            const fileIDs = [];

            for (const fileInfo of uploadedFiles.value) {
                const formData = new FormData();
                formData.append('file', fileInfo.file);
                formData.append('session_id', sessionID);

                try {
                    const response = await axios.post(`${API_BASE}/files/upload`, formData, {
                        headers: {
                            'Content-Type': 'multipart/form-data'
                        }
                    });

                    if (response.data.file_id) {
                        fileIDs.push(response.data.file_id);
                    }
                } catch (error) {
                    console.error('文件上传失败:', error);
                    alert(`文件上传失败: ${fileInfo.name}`);
                }
            }

            return fileIDs;
        };

        // 获取模型列表
        const fetchModels = async () => {
            try {
                const response = await axios.get(`${API_BASE}/models`);
                models.value = response.data.data;
            } catch (error) {
                console.error('获取模型列表失败:', error);
            }
        };

        // 获取会话列表
        const fetchSessions = async () => {
            try {
                const response = await axios.get(`${API_BASE}/chat/sessions`);
                sessions.value = response.data.data;
            } catch (error) {
                console.error('获取会话列表失败:', error);
            }
        };

        // 获取特定会话的消息
        const fetchSessionMessages = async (sessionID) => {
            try {
                const response = await axios.get(`${API_BASE}/chat/sessions/${sessionID}/messages`);
                return response.data.data;
            } catch (error) {
                console.error('获取会话消息失败:', error);
                return [];
            }
        };

        // 创建新会话
        const createNewSession = async () => {
            if (!selectedModel.value) return;

            try {
                const response = await axios.post(`${API_BASE}/chat/session`, {
                    model_name: selectedModel.value,
                    persona: selectedPersona.value
                });

                sessionId.value = response.data.session_id;
                messages.value = [];

                // 刷新会话列表
                await fetchSessions();
                console.log('新会话创建成功:', sessionId.value);
            } catch (error) {
                console.error('创建会话失败:', error);
                alert('创建会话失败: ' + (error.response?.data?.error || error.message));
            }
        };

        // 切换会话
        const switchSession = async (session) => {
            if (session.SessionID === sessionId.value) return;

            sessionId.value = session.SessionID;
            selectedModel.value = session.ModelName;

            // 加载该会话的消息
            const sessionMessages = await fetchSessionMessages(session.SessionID);
            messages.value = sessionMessages.map((msg, index) => ({
                id: Date.now() + index,
                role: msg.role,
                content: msg.content,
                streaming: false
            }));

            scrollToBottom();
        };

        // 删除会话
        const deleteSession = async (sessionID) => {
            if (!confirm('确定要删除这个会话吗？此操作不可恢复。')) {
                return;
            }

            try {
                await axios.delete(`${API_BASE}/chat/sessions/${sessionID}`);

                // 如果删除的是当前会话，清空当前会话
                if (sessionID === sessionId.value) {
                    sessionId.value = '';
                    messages.value = [];
                }

                // 刷新会话列表
                await fetchSessions();
            } catch (error) {
                console.error('删除会话失败:', error);
                alert('删除会话失败: ' + (error.response?.data?.error || error.message));
            }
        };

        // 发送消息（流式版本）- 修正SSE解析
        const sendMessage = async () => {
            if ((!inputMessage.value.trim() && uploadedFiles.value.length === 0) || !sessionId.value || isTyping.value) return;

            const userMessage = inputMessage.value.trim();

            // 上传文件并获取文件ID
            let fileIDs = [];
            if (uploadedFiles.value.length > 0) {
                fileIDs = await uploadFiles(sessionId.value);
            }

            inputMessage.value = '';

            // 添加用户消息到界面
            messages.value.push({
                id: Date.now(),
                role: 'user',
                content: userMessage + (uploadedFiles.value.length > 0 ? ` [附件: ${uploadedFiles.value.map(f => f.name).join(', ')}]` : ''),
                streaming: false
            });

            scrollToBottom();
            isTyping.value = true;

            // 创建AI消息项，初始内容为空
            const aiMessageId = Date.now() + 1;
            const aiMessage = {
                id: aiMessageId,
                role: 'assistant',
                content: '',
                streaming: true
            };
            messages.value.push(aiMessage);

            scrollToBottom();

            try {
                const response = await fetch(`${API_BASE}/chat/message/stream`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        session_id: sessionId.value,
                        model_name: selectedModel.value,
                        message: userMessage,
                        persona: selectedPersona.value,
                        file_ids: fileIDs  // 添加文件ID
                    })
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                const reader = response.body.getReader();
                const decoder = new TextDecoder();

                // 用于累积不完整的数据
                let buffer = '';

                while (true) {
                    const { value, done } = await reader.read();
                    if (done) break;

                    // 解码并添加到缓冲区
                    buffer += decoder.decode(value, { stream: true });

                    // 按行分割
                    const lines = buffer.split('\n');
                    // 保留最后一行（可能不完整）
                    buffer = lines.pop() || '';

                    for (const line of lines) {
                        if (line.startsWith('data: ')) {
                            const dataStr = line.slice(6);
                            if (dataStr.trim() === '') continue;

                            try {
                                const data = JSON.parse(dataStr);

                                if (data.content !== undefined) {
                                    // 更新AI消息内容
                                    const aiMsg = messages.value.find(msg => msg.id === aiMessageId);
                                    if (aiMsg) {
                                        aiMsg.content += data.content;
                                    }
                                    scrollToBottom();
                                }

                                if (data.done) {
                                    // 流式传输完成
                                    const aiMsg = messages.value.find(msg => msg.id === aiMessageId);
                                    if (aiMsg) {
                                        aiMsg.streaming = false;
                                    }
                                    break;
                                }

                                if (data.error) {
                                    throw new Error(data.error);
                                }
                            } catch (e) {
                                console.error('解析SSE数据失败:', e, '数据:', dataStr);
                            }
                        }
                    }
                }

                // 清空已上传文件列表
                uploadedFiles.value = [];
                // 刷新会话列表以更新消息计数和最后更新时间
                await fetchSessions();
            } catch (error) {
                console.error('发送消息失败:', error);
                // 更新AI消息为错误信息
                const aiMsg = messages.value.find(msg => msg.id === aiMessageId);
                if (aiMsg) {
                    aiMsg.content = '抱歉，发生了错误: ' + error.message;
                    aiMsg.streaming = false;
                }
                scrollToBottom();
            } finally {
                isTyping.value = false;
            }
        };

        // 格式化日期
        const formatDate = (dateString) => {
            const date = new Date(dateString);
            const now = new Date();
            const diffMs = now - date;
            const diffMins = Math.floor(diffMs / (1000 * 60));
            const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
            const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

            if (diffMins < 1) return '刚刚';
            if (diffMins < 60) return `${diffMins}分钟前`;
            if (diffHours < 24) return `${diffHours}小时前`;
            if (diffDays < 7) return `${diffDays}天前`;

            return date.toLocaleDateString();
        };

        // 滚动到底部
        const scrollToBottom = () => {
            nextTick(() => {
                if (chatContainer.value) {
                    chatContainer.value.scrollTop = chatContainer.value.scrollHeight;
                }
            });
        };

        const fetchPersonas = async () => {
            try {
                const response = await axios.get(`${API_BASE}/personas`);
                personas.value = response.data.data;
            } catch (error) {
                console.error('获取人格列表失败:', error);
            }
        };

        onMounted(() => {
            fetchModels();
            fetchSessions();
            fetchPersonas();
            configureMarkdown();
            // 调试：检查fileInput是否正确绑定
            console.log('fileInput ref:', fileInput.value);
        });

        return {
            models,
            sessions,
            selectedModel,
            sessionId,
            messages,
            inputMessage,
            isTyping,
            chatContainer,
            createNewSession,
            switchSession,
            deleteSession,
            sendMessage,
            formatDate,
            personas,
            selectedPersona,
            uploadedFiles,
            fileInput,
            triggerFileInput,
            handleFileUpload,
            removeFile,
            renderMarkdown
        };
    }
}).mount('#app');