import re
import sys

def update_bot_service_file(filename):
    with open(filename, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Patterns to update method signatures
    # These methods need to accept *tgbotapi.Message instead of string/no params
    updates = [
        # Methods that take string -> should take *tgbotapi.Message
        (r'func \(s \*BotService\) handleImportCommand\(text string\)', 'func (s *BotService) handleImportCommand(msg *tgbotapi.Message)'),
        (r'func \(s \*BotService\) handleTrainCommand\(text string\)', 'func (s *BotService) handleTrainCommand(msg *tgbotapi.Message)'),
        (r'func \(s \*BotService\) handlePredictCommand\(text string\)', 'func (s *BotService) handlePredictCommand(msg *tgbotapi.Message)'),
        (r'func \(s \*BotService\) handleAddPositionCommand\(text string\)', 'func (s *BotService) handleAddPositionCommand(msg *tgbotapi.Message)'),
        (r'func \(s \*BotService\) handleEditPositionCommand\(text string\)', 'func (s *BotService) handleEditPositionCommand(msg *tgbotapi.Message)'),
        
        # Methods that take no params -> should take *tgbotapi.Message
        (r'func \(s \*BotService\) handleTimeCommand\(\)', 'func (s *BotService) handleTimeCommand(msg *tgbotapi.Message)'),
        (r'func \(s \*BotService\) handleStatusCommand\(\)', 'func (s *BotService) handleStatusCommand(msg *tgbotapi.Message)'),
        (r'func \(s \*BotService\) handleRiskCommand\(\)', 'func (s *BotService) handleRiskCommand(msg *tgbotapi.Message)'),
        (r'func \(s \*BotService\) handleLimitsCommand\(\)', 'func (s *BotService) handleLimitsCommand(msg *tgbotapi.Message)'),
        (r'func \(s \*BotService\) handlePositionsCommand\(\)', 'func (s *BotService) handlePositionsCommand(msg *tgbotapi.Message)'),
        (r'func \(s \*BotService\) handleRestartCommand\(\)', 'func (s *BotService) handleRestartCommand(msg *tgbotapi.Message)'),
        (r'func \(s \*BotService\) handleRestartOTP\(otp string\)', 'func (s *BotService) handleRestartOTP(chatID int64, otp string)'),
    ]
    
    for pattern, replacement in updates:
        content = re.sub(pattern, replacement, content)
    
    # Now update inside the method bodies to use msg.Text and msg.Chat.ID
    # This is more complex - we need to add these lines at the beginning of each method
    
    with open(filename, 'w', encoding='utf-8') as f:
        f.write(content)
    
    print(f"Updated {filename}")

if __name__ == "__main__":
    update_bot_service_file("internal/service/telegram/bot_service.go")
