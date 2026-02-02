import re

def fix_bot_service():
    filepath = "internal/service/telegram/bot_service.go"
    with open(filepath, 'r', encoding='utf-8') as f:
        lines = f.readlines()
    
    output = []
    inside_method = False
    method_name = None
    indent_level = 0
    chatid_added = False
    text_added = False
    
    for i, line in enumerate(lines):
        # Detect method start
        method_match = re.match(r'^func \(s \*BotService\) (handle\w+)\(msg \*tgbotapi\.Message\)', line)
        if method_match:
            inside_method = True
            method_name = method_match.group(1)
            output.append(line)
            chatid_added = False
            text_added = False
            continue
        
        # If we're inside a method and see the first {
        if inside_method and not chatid_added and '{' in line and method_name:
            output.append(line)
            # Add chatID extraction after the opening brace
            indent = '\t'
            output.append(f'{indent}chatID := msg.Chat.ID\n')
            # Check if we need text
            if 'text' in method_name.lower() or method_name in ['handleImportCommand', 'handleTrainCommand', 'handlePredictCommand', 'handleAddPositionCommand', 'handleEditPositionCommand']:
                output.append(f'{indent}text := msg.Text\n')
                text_added = True
            chatid_added = True
            continue
        
        # Replace SendMessage calls
        if inside_method:
            # Count braces to know when method ends
            indent_level += line.count('{') - line.count('}')
            
            if indent_level <= 0:
                inside_method = False
                method_name = None
                chatid_added = False
                text_added = False
            
            # Replace s.SendMessage("... to s.SendMessage(chatID, "...
            if 's.SendMessage("' in line:
                line = line.replace('s.SendMessage("', 's.SendMessage(chatID, "')
        
        output.append(line)
    
    with open(filepath, 'w', encoding='utf-8') as f:
        f.writelines(output)
    
    print(f"Fixed {filepath}")

if __name__ == "__main__":
    fix_bot_service()
