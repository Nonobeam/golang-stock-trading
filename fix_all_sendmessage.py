import re
import os

def fix_file_sendmessage(filepath, has_msg_param=False):
    """
    Fix SendMessage calls in a file
    If has_msg_param=True, assumes methods take msg parameter and extracts chatID
    If False, looks for existing chatID or context
    """
    if not os.path.exists(filepath):
        print(f"Skipping {filepath} - not found")
        return
    
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # For files with methods that accept msg *tgbotapi.Message parameter
    if has_msg_param:
        # Update method signatures to accept *tgbotapi.Message
        content = re.sub(
            r'func \(s \*BotService\) (handleAddPositionCommand|handleEditPositionCommand)\(text string\)',
            r'func (s *BotService) \1(msg *tgbotapi.Message)',
            content
        )
        
        # Find methods and add chatID/text extraction
        def add_extraction(match):
            method_sig = match.group(0)
            # Check if chatID is already declared
            if 'chatID := msg.Chat.ID' not in match.group(0):
                return method_sig + '\n\tchatID := msg.Chat.ID\n\ttext := msg.Text'
            return method_sig
        
        # Add after method signature line for position commands
        lines = content.split('\n')
        new_lines = []
        for i, line in enumerate(lines):
            new_lines.append(line)
            if re.match(r'^func \(s \*BotService\) (handleAddPositionCommand|handleEditPositionCommand)\(msg \*tgbotapi\.Message\) \{', line):
                # Check if next line already has chatID
                if i+1 < len(lines) and 'chatID' not in lines[i+1]:
                    new_lines.append('\tchatID := msg.Chat.ID')
                    new_lines.append('\ttext := msg.Text')
        content = '\n'.join(new_lines)
    
    # Replace all s.SendMessage(" with s.SendMessage(chatID, "
    content = re.sub(r's\.SendMessage\("', r's.SendMessage(chatID, "', content)
    content = re.sub(r's\.SendMessage\(fmt\.Sprintf', r's.SendMessage(chatID, fmt.Sprintf', content)
    
    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(content)
    
    print(f"Fixed {filepath}")

def fix_alert_service():
    """Fix alert_service.go which uses a.botService.SendMessage"""
    filepath = "internal/service/telegram/alert_service.go"
    if not os.path.exists(filepath):
        return
    
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # These methods need chatID parameter added
    # For now, we'll comment them out or add a TODO
    # Better: Add chatID parameter to each notification method
    
    # Replace a.botService.SendMessage(" with a.botService.SendMessage(chatID, "
    content = re.sub(r'a\.botService\.SendMessage\("', r'a.botService.SendMessage(chatID, "', content)
    content = re.sub(r'a\.botService\.SendMessage\(fmt\.Sprintf', r'a.botService.SendMessage(chatID, fmt.Sprintf', content)
    
    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(content)
    
    print(f"Fixed {filepath}")

# Fix all the files
print("Fixing bot service files...")
fix_file_sendmessage("internal/service/telegram/bot_service_positions.go", has_msg_param=True)
fix_file_sendmessage("internal/service/telegram/bot_service_alerts.go", has_msg_param=False)
fix_alert_service()

print("Done!")
