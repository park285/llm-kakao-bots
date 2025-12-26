package adapter

import (
	"strings"

	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// MemberDirectoryGroup: 멤버 목록 표시를 위한 그룹 (예: 'JP 3기생', 'EN Promise')
type MemberDirectoryGroup struct {
	GroupName string
	Members   []MemberDirectoryEntry
}

// MemberDirectoryEntry: 멤버 목록의 개별 항목 (주 이름 및 보조 이름 포함)
type MemberDirectoryEntry struct {
	PrimaryName   string
	SecondaryName string
}

type memberDirectoryTemplateData struct {
	Emoji  UIEmoji
	Total  int
	Groups []memberDirectoryGroupView
}

type memberDirectoryGroupView struct {
	GroupName string
	Members   []memberDirectoryEntryView
}

type memberDirectoryEntryView struct {
	Primary   string
	Secondary string
	ShowBoth  bool
}

// MemberDirectory: 전체 멤버 디렉토리 목록을 포맷팅하여 메시지 문자열을 생성한다.
func (f *ResponseFormatter) MemberDirectory(groups []MemberDirectoryGroup, total int) string {
	viewGroups := prepareMemberDirectoryGroups(groups)

	if total <= 0 {
		for _, group := range viewGroups {
			total += len(group.Members)
		}
	}

	data := memberDirectoryTemplateData{
		Emoji:  DefaultEmoji,
		Total:  total,
		Groups: viewGroups,
	}

	rendered, err := executeFormatterTemplate("member_directory.tmpl", data)
	if err != nil {
		return ErrorMessage(ErrDisplayMemberListFailed)
	}

	if len(viewGroups) == 0 {
		return rendered
	}

	instruction, body := splitTemplateInstruction(rendered)
	if instruction == "" || body == "" {
		return rendered
	}
	return util.ApplyKakaoSeeMorePadding(body, instruction)
}

func prepareMemberDirectoryGroups(groups []MemberDirectoryGroup) []memberDirectoryGroupView {
	if len(groups) == 0 {
		return nil
	}

	views := make([]memberDirectoryGroupView, 0, len(groups))
	for _, group := range groups {
		name := util.TrimSpace(group.GroupName)
		if name == "" {
			name = "기타"
		}

		members := make([]memberDirectoryEntryView, 0, len(group.Members))
		for _, member := range group.Members {
			primary := util.TrimSpace(member.PrimaryName)
			secondary := util.TrimSpace(member.SecondaryName)
			if primary == "" && secondary == "" {
				continue
			}

			entry := memberDirectoryEntryView{
				Primary:   primary,
				Secondary: secondary,
				ShowBoth:  primary != "" && secondary != "" && !strings.EqualFold(primary, secondary),
			}
			members = append(members, entry)
		}

		if len(members) == 0 {
			continue
		}

		views = append(views, memberDirectoryGroupView{
			GroupName: name,
			Members:   members,
		})
	}

	return views
}
