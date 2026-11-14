// Vạn Linh Bội teaches one missing HMLV skill without overwriting occupied slots.
package pet

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

func (s *Service) UseHMLVPendant(ctx context.Context, charID, petID, itemID int64) (*domainpet.Pet, *domainitem.Item, string, error) {
	petMutationMu.Lock()
	defer petMutationMu.Unlock()
	if s.itemRepo == nil || s.petRepo == nil || s.gameDataManager == nil {
		return nil, nil, "", pkgerrors.ErrSystemError
	}
	source, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, nil, "", err
	}
	if source.CharacterID != charID || source.TemplateID != 6859 || source.StackCount <= 0 || (source.SlotType != domainitem.SlotTypeBag && source.SlotType != domainitem.SlotTypePetItemBag) {
		return nil, nil, "", pkgerrors.ErrInvalidInput
	}
	p, err := s.getPetAndValidate(ctx, charID, petID)
	if err != nil {
		return nil, nil, "", err
	}
	if p.TemplateID != 2281 && p.TemplateID != 2282 {
		return nil, source, "Vạn Linh Bội chỉ dùng cho Huyền Minh Linh Vũ; không tiêu hao vật phẩm.", nil
	}
	original := clonePetSnapshot(p)
	next := clonePetSnapshot(p)
	ensurePetSkillSlots(next)
	candidates := []int{}
	for _, id := range []int{7011, 7014} {
		skill := s.gameDataManager.GetSkill(id)
		if skill == nil {
			return nil, nil, "", pkgerrors.ErrSystemError
		}
		if !s.petHasConflictingSkill(next, skill) {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 0 {
		return nil, source, "Pet đã có Thợ Săn Sát Thủ và Tứ Phương Khiếp Đảm; không tiêu hao Vạn Linh Bội.", nil
	}
	slot, ok := firstAvailablePetSkillSlot(next)
	if !ok {
		return nil, source, "Pet không có ô kỹ năng trống đã mở; không tiêu hao Vạn Linh Bội.", nil
	}
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(candidates))))
	if err != nil {
		return nil, nil, "", err
	}
	skillID := candidates[int(index.Int64())]
	setPetSkillSlotValue(next, slot, skillID)
	s.syncPetSkillBonuses(next)
	next.RecalculateStats()
	if err := s.petRepo.Save(ctx, next); err != nil {
		return nil, nil, "", err
	}
	remaining, err := s.consumePetSkillBookItem(ctx, source)
	if err != nil {
		return nil, nil, "", errors.Join(err, s.petRepo.Save(ctx, original))
	}
	consumed := *source
	consumed.StackCount = remaining
	return next, &consumed, "Pet đã học " + s.gameDataManager.GetSkill(skillID).Name + ".", nil
}
