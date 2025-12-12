package commands

// func TestCallbackHandler_HandleCallback_ParsesSessionIDCorrectly(t *testing.T) {
// 	// Create mock DB manager
// 	mockDB := new(MockDBManager)

// 	// Setup expected calls
// 	mockDB.On("IsSessionOwner", mock.Anything, 123, int64(456)).Return(true, nil)

// 	// Create handler
// 	handler := NewCallbackHandler(mockDB)

// 	// Create test callback
// 	callback := &tgbotapi.CallbackQuery{
// 		ID: "test_callback_id",
// 		From: &tgbotapi.User{
// 			ID: 456,
// 		},
// 		Message: &tgbotapi.Message{
// 			Chat: &tgbotapi.Chat{
// 				ID: 789,
// 			},
// 			MessageID: 101,
// 		},
// 		Data: "confirm_task:123", // Using the format: action:sessionid
// 	}

// 	// Handle callback
// 	response := handler.HandleCallback(callback)

// 	// Assert response
// 	assert.NotNil(t, response)
// 	assert.True(t, response.IsOwner)
// 	assert.NotNil(t, response.CallbackConfig)

// 	// Verify mock was called with correct session ID
// 	mockDB.AssertCalled(t, "IsSessionOwner", mock.Anything, 123, int64(456))
// }

// func TestCallbackHandler_HandleCallback_NonOwner(t *testing.T) {
// 	// Create mock DB manager
// 	mockDB := new(MockDBManager)

// 	// Setup expected calls - return false for ownership
// 	mockDB.On("IsSessionOwner", mock.Anything, 123, int64(456)).Return(false, nil)

// 	// Create handler
// 	handler := NewCallbackHandler(mockDB)

// 	// Create test callback
// 	callback := &tgbotapi.CallbackQuery{
// 		ID: "test_callback_id",
// 		From: &tgbotapi.User{
// 			ID: 456,
// 		},
// 		Message: &tgbotapi.Message{
// 			Chat: &tgbotapi.Chat{
// 				ID: 789,
// 			},
// 			MessageID: 101,
// 		},
// 		Data: "cancel_task:123", // Using the format: action:sessionid
// 	}

// 	// Handle callback
// 	response := handler.HandleCallback(callback)

// 	// Assert response
// 	assert.NotNil(t, response)
// 	assert.False(t, response.IsOwner)
// 	assert.NotNil(t, response.CallbackConfig)
// 	assert.Contains(t, response.CallbackConfig.Text, "Only the user who started this discussion")
// }

// func TestCallbackHandler_HandleCallback_InvalidCallbackData(t *testing.T) {
// 	// Create mock DB manager
// 	mockDB := new(MockDBManager)

// 	// Create handler
// 	handler := NewCallbackHandler(mockDB)

// 	// Create test callback with invalid data
// 	callback := &tgbotapi.CallbackQuery{
// 		ID: "test_callback_id",
// 		From: &tgbotapi.User{
// 			ID: 456,
// 		},
// 		Data: "invalid_format",
// 	}

// 	// Handle callback
// 	response := handler.HandleCallback(callback)

// 	// Assert response
// 	assert.NotNil(t, response)
// 	assert.False(t, response.IsOwner)
// 	assert.NotNil(t, response.CallbackConfig)
// 	assert.Contains(t, response.CallbackConfig.Text, "Invalid callback data")

// 	// The mock should not have been called
// 	mockDB.AssertNotCalled(t, "IsSessionOwner", mock.Anything, mock.Anything, mock.Anything)
// }
